package workqueue_test

import (
	"os"
	"testing"
	"time"

	"github.com/abustany/moblog-cloud/pkg/workqueue"
)

func TestRedisQueue(t *testing.T) {
	// Reduce groom interval at expense of CPU load so that tests don't take
	// forever to run
	workqueue.GroomInterval = 20 * time.Millisecond

	redisURL := os.Getenv("REDIS_URL")

	if redisURL == "" {
		t.Skip("REDIS_URL environment variable not defined")
	}

	q, err := workqueue.NewRedisQueue(redisURL)

	if err != nil {
		t.Fatalf("Error while creating queue: %s", err)
	}

	defer q.Stop()

	if err := q.Clear(); err != nil {
		t.Fatalf("Error while clearing queue: %s", err)
	}

	testWorkqueue(t, q)
}
