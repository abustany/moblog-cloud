package workqueue

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"

	"github.com/abustany/moblog-cloud/pkg/distlock"
	"github.com/abustany/moblog-cloud/pkg/idgenerator"
)

const redisKeyPending = "jobs-pending"
const redisKeyReserved = "jobs-reserved"

type RedisQueue struct {
	client          *redis.Client
	id              string
	idGenerator     *idgenerator.StringIdGenerator
	finishScriptSha string
	groomScriptSha  string
	groomLock       *distlock.Lock
	groomTicker     *time.Ticker
}

type redisEntry struct {
	ID    string
	TTRus int64
	Data  []byte // job data encoded with gob
}

func NewRedisQueue(redisURL string) (*RedisQueue, error) {
	options, err := redis.ParseURL(redisURL)

	if err != nil {
		return nil, errors.Wrap(err, "Error while parsing redis URL")
	}

	hostname, err := os.Hostname()

	if err != nil {
		return nil, errors.Wrap(err, "Error while retrieving hostname")
	}

	client := redis.NewClient(options)

	finishScriptSha, err := client.ScriptLoad(finishScript).Result()

	if err != nil {
		return nil, errors.Wrap(err, "Error while uploading finish script to Redis")
	}

	groomScriptSha, err := client.ScriptLoad(groomScript).Result()

	if err != nil {
		return nil, errors.Wrap(err, "Error while uploading groom script to Redis")
	}

	groomLock, err := distlock.New(distlock.Options{
		Name:   "jobs",
		Client: client,
	})

	if err != nil {
		return nil, errors.Wrap(err, "Error while creating grooming lock")
	}

	q := &RedisQueue{
		client:          client,
		id:              hostname + "-" + uuid.NewV4().String(),
		idGenerator:     &idgenerator.StringIdGenerator{},
		finishScriptSha: finishScriptSha,
		groomScriptSha:  groomScriptSha,
		groomLock:       groomLock,
		groomTicker:     time.NewTicker(GroomInterval),
	}

	go func() {
		for range q.groomTicker.C {
			q.groom()
		}
	}()

	return q, nil
}

func (q *RedisQueue) Stop() {
	q.groomTicker.Stop()
	q.groomLock.Stop()
}

func encodeRedisEntry(id string, ttr time.Duration, data interface{}) ([]byte, error) {
	encodedData := bytes.Buffer{}

	if err := gob.NewEncoder(&encodedData).Encode(&data); err != nil {
		return nil, errors.Wrap(err, "Error while encoding entry data")
	}

	encodedEntry, err := json.Marshal(&redisEntry{
		ID:    id,
		TTRus: int64(ttr / time.Microsecond),
		Data:  encodedData.Bytes(),
	})

	return encodedEntry, errors.Wrap(err, "Error while encoding entry")
}

func decodeRedisEntry(data []byte) (*JobEntry, error) {
	var decodedRedisEntry redisEntry

	if err := json.Unmarshal(data, &decodedRedisEntry); err != nil {
		return nil, errors.Wrap(err, "Error while decoding entry")
	}

	decodedEntry := JobEntry{
		ID:  decodedRedisEntry.ID,
		TTR: time.Duration(decodedRedisEntry.TTRus) * time.Microsecond,
	}

	if err := gob.NewDecoder(bytes.NewReader(decodedRedisEntry.Data)).Decode(&decodedEntry.Data); err != nil {
		return nil, errors.Wrap(err, "Error while decoding entry data")
	}

	return &decodedEntry, nil
}

func (q *RedisQueue) Post(job interface{}, ttr time.Duration) error {
	entryID := q.id + "-" + q.idGenerator.Next()
	data, err := encodeRedisEntry(entryID, ttr, job)

	if err != nil {
		return errors.Wrap(err, "Error while encoding entry data")
	}

	return errors.Wrap(q.client.LPush(redisKeyPending, data).Err(), "Error while pushing job to Redis")
}

func (q *RedisQueue) Pick(timeout time.Duration) (*JobEntry, error) {
	if timeout < time.Second {
		// Redis cannot wait less than a second
		timeout = time.Second
	}

	entryData, err := q.client.BRPopLPush(redisKeyPending, redisKeyReserved, timeout).Bytes()

	if err == redis.Nil {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "Error while picking job from Redis")
	}

	return decodeRedisEntry([]byte(entryData))
}

const finishScript = `
local reservedList = KEYS[1]
local entryId = ARGV[1]
local entries = redis.call('lrange', reservedList, 0, -1)
local finished = 'finished'

for i, entryData in ipairs(entries) do
	local entry = cjson.decode(entryData)

	if entry.ID == entryId then
		redis.call('lset', reservedList, i-1, finished)
	end
end

redis.call('lrem', reservedList, 1, finished)
`

func (q *RedisQueue) Finish(entry *JobEntry) error {
	if err := q.client.EvalSha(q.finishScriptSha, []string{redisKeyReserved}, entry.ID).Err(); err != nil && err != redis.Nil {
		return errors.Wrap(err, "Error while removing entry from reserved list")
	}

	return nil
}

func (q *RedisQueue) Clear() error {
	return errors.Wrap(q.client.Del(redisKeyPending, redisKeyReserved).Err(), "Error while deleting keys")
}

const groomScript = `
local pendingList = KEYS[1]
local reservedList = KEYS[2]
local redisTime = redis.call('time')
local time = 1000000*redisTime[1]+redisTime[2]
local entries = redis.call('lrange', reservedList, 0, -1)
local expired = 'expired'

for i, entryData in ipairs(entries) do
	local entry = cjson.decode(entryData)

    if entry.Started == nil then
		entry.Started = time
		redis.call('lset', reservedList, i-1, cjson.encode(entry))
	end
	
	if time > (entry.Started + entry.TTRus) then
		redis.call('lset', reservedList, i-1, expired)
		entry.Started = nil
		redis.call('lpush', pendingList, cjson.encode(entry))
	end
end

redis.call('lrem', reservedList, 0, expired)
`

func (q *RedisQueue) groom() {
	if !q.groomLock.IsMaster() {
		return
	}

	if err := q.client.EvalSha(q.groomScriptSha, []string{redisKeyPending, redisKeyReserved}).Err(); err != nil && err != redis.Nil {
		log.Printf("Error while running groom script: %s", err)
	}
}
