package workqueue_test

import (
	"testing"
	"time"

	"github.com/abustany/moblog-cloud/pkg/workqueue"
)

func TestMemoryQueue(t *testing.T) {
	// Reduce groom interval at expense of CPU load so that tests don't take
	// forever to run
	workqueue.MemoryGroomInterval = 20 * time.Millisecond

	q, err := workqueue.NewMemoryQueue()

	if err != nil {
		t.Fatalf("Error while creating queue: %s", err)
	}

	defer q.Stop()

	testWorkqueue(t, q)
}
