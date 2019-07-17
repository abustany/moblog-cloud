package distlock

import (
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

type Lock struct {
	options          Options
	id               string
	ticker           *time.Ticker
	acquireScriptSha string
	releaseScriptSha string
	master           uint32
	masterChannel    chan bool
}

type Options struct {
	Name            string
	Client          *redis.Client
	RefreshInterval time.Duration
	ExpirationDelay time.Duration
}

const DefaultRefreshInterval = 5 * time.Second
const DefaultExpirationDelay = 5 * DefaultRefreshInterval

func DefaultOptions(redisURL, name string) (Options, error) {
	clientOptions, err := redis.ParseURL(redisURL)

	if err != nil {
		return Options{}, errors.Wrap(err, "Error while parsing Redis URL")
	}

	return Options{
		Name:            name,
		Client:          redis.NewClient(clientOptions),
		RefreshInterval: DefaultRefreshInterval,
		ExpirationDelay: DefaultExpirationDelay,
	}, nil
}

const acquireScript = `
local lockKey = KEYS[1]
local nodeId = ARGV[1]
local expirationDelayMs = ARGV[2]
local masterId = redis.call('get', lockKey)

if masterId == false then
  -- No master yet, we become the master
  redis.call('set', lockKey, nodeId, 'PX', expirationDelayMs)
  return 1
end

if masterId == nodeId then
  -- We're the master already, refresh our lease
  redis.call('pexpire', lockKey, expirationDelayMs)
  return 1
end

-- Somebody else is the master, don't touch anything
return 0
`

const releaseScript = `
local lockKey = KEYS[1]
local nodeId = ARGV[1]
local masterId = redis.call('get', lockKey)

if masterId == nodeId then
  redis.call('del', lockKey)
  return 1
end

return 0
`

func New(options Options) (*Lock, error) {
	if options.Name == "" {
		return nil, errors.New("Lock name cannot be empty")
	}

	if options.RefreshInterval == 0 {
		options.RefreshInterval = DefaultRefreshInterval
	}

	if options.ExpirationDelay == 0 {
		options.ExpirationDelay = DefaultExpirationDelay
	}

	hostname, err := os.Hostname()

	if err != nil {
		return nil, errors.Wrap(err, "Error while getting hostname")
	}

	acquireScriptSha, err := options.Client.ScriptLoad(acquireScript).Result()

	if err != nil {
		return nil, errors.Wrap(err, "Error while loading lock acquire script")
	}

	releaseScriptSha, err := options.Client.ScriptLoad(releaseScript).Result()

	if err != nil {
		return nil, errors.Wrap(err, "Error while loading lock release script")
	}

	l := &Lock{
		options:          options,
		id:               hostname + "-" + uuid.NewV4().String(),
		ticker:           time.NewTicker(options.RefreshInterval),
		acquireScriptSha: acquireScriptSha,
		releaseScriptSha: releaseScriptSha,
		master:           0,
		masterChannel:    make(chan bool, 1),
	}

	go func() {
		for range l.ticker.C {
			l.tick()
		}
	}()

	return l, nil
}

func (l *Lock) Stop() {
	err := l.options.Client.EvalSha(l.releaseScriptSha, []string{l.options.Name}, l.id).Err()

	if err != nil {
		log.Printf("Error while releasing the lock: %s", err)
	}

	l.ticker.Stop()
	atomic.StoreUint32(&l.master, 0)

	if l.masterChannel != nil {
		close(l.masterChannel)
		l.masterChannel = nil
	}
}

func (l *Lock) ID() string {
	return l.id
}

func (l *Lock) IsMaster() bool {
	return atomic.LoadUint32(&l.master) == 1
}

func (l *Lock) MasterChannel() <-chan bool {
	return l.masterChannel
}

func (l *Lock) tick() {
	ret, err := l.options.Client.EvalSha(l.acquireScriptSha, []string{l.options.Name}, l.id, int64(l.options.ExpirationDelay/time.Millisecond)).Int()

	if err != nil {
		log.Printf("Error while refreshing lock: %s", err)
		return
	}

	master := ret == 1

	changed := atomic.SwapUint32(&l.master, uint32(ret)) != uint32(ret)

	if changed {
		select {
		case l.masterChannel <- master:
		default:
		}
	}
}
