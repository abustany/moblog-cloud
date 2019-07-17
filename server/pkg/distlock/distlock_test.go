package distlock_test

import (
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis"

	"github.com/abustany/moblog-cloud/pkg/distlock"
)

const refreshInterval = 50 * time.Millisecond

func makeLock(t *testing.T, name string) *distlock.Lock {
	redisURL := os.Getenv("REDIS_URL")

	if redisURL == "" {
		t.Skip("REDIS_URL environment variable not defined")
	}

	redisOptions, err := redis.ParseURL(redisURL)

	if err != nil {
		t.Fatalf("Error while parsing Redis URL: %s", err)
	}

	options := distlock.Options{
		Client:          redis.NewClient(redisOptions),
		Name:            name,
		RefreshInterval: refreshInterval,
		ExpirationDelay: 5 * refreshInterval,
	}

	lock, err := distlock.New(options)

	if err != nil {
		t.Fatalf("Error while creating lock: %s", err)
	}

	return lock
}

func waitForMaster(t *testing.T, lock *distlock.Lock) {
	select {
	case <-time.After(10 * refreshInterval):
		t.Errorf("Master signal didn't come for lock %s", lock.ID())
	case master := <-lock.MasterChannel():
		if !master {
			t.Errorf("Master channel didn't send true for lock %s", lock.ID())
		}

		if !lock.IsMaster() {
			t.Errorf("IsMaster returns false for lock %s", lock.ID())
		}
	}
}

func TestOneLock(t *testing.T) {
	lock := makeLock(t, "lock")
	defer lock.Stop()

	waitForMaster(t, lock)
}

func TestTwoLocks(t *testing.T) {
	lock1 := makeLock(t, "lock")
	defer lock1.Stop()

	waitForMaster(t, lock1)

	lock2 := makeLock(t, "lock")
	defer lock2.Stop()

	select {
	case <-time.After(5 * refreshInterval):
	case <-lock2.MasterChannel():
		t.Errorf("Lock 2 became master while lock 1 is still running")
	}

	lock1.Stop()
	waitForMaster(t, lock2)
}
