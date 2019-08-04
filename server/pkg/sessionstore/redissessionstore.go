package sessionstore

import (
	"encoding/json"
	"time"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"
)

const redisKeyPrefixSessions = "session-"

type RedisSessionStore struct {
	client *redis.Client
}

func NewRedisSessionStore(redisURL string) (*RedisSessionStore, error) {
	options, err := redis.ParseURL(redisURL)

	if err != nil {
		return nil, errors.Wrap(err, "Error while parsing redis URL")
	}

	return &RedisSessionStore{redis.NewClient(options)}, nil
}

func (s *RedisSessionStore) Set(session Session) error {
	data, err := json.Marshal(&session)

	if err != nil {
		return errors.Wrap(err, "Error while encoding session data")
	}

	if err := s.client.Set(redisKeyPrefixSessions+session.Sid, data, session.Expires.Sub(time.Now())).Err(); err != nil {
		return errors.Wrap(err, "Error while saving session into Redis")
	}

	return nil
}

func (s *RedisSessionStore) Get(sid string) (*Session, error) {
	data, err := s.client.Get(redisKeyPrefixSessions + sid).Bytes()

	if err == redis.Nil {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "Error while retrieving session data from Redis")
	}

	var session Session

	if err := json.Unmarshal(data, &session); err != nil {
		return nil, errors.Wrap(err, "Error while decoding session data")
	}

	if time.Now().After(session.Expires) {
		// Just in case Redis hasn't had time to clean it up yet
		return nil, nil
	}

	return &session, nil
}

func (s *RedisSessionStore) Delete(sid string) error {
	return errors.Wrap(s.client.Del(redisKeyPrefixSessions+sid).Err(), "Error while deleting session from Redis")
}
