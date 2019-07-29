package workqueue_test

import (
	"testing"
	"time"

	"github.com/abustany/moblog-cloud/pkg/workqueue"
)

func testWorkqueue(t *testing.T, q workqueue.Queue) {
	withQueue := func(f func(*testing.T, workqueue.Queue)) func(*testing.T) {
		return func(t *testing.T) {
			f(t, q)
		}
	}

	t.Run("Pick on an empty queue", withQueue(testPickEmpty))
	t.Run("Post a job and pick it", withQueue(testPostPick))
	t.Run("Pick times out", withQueue(testPickTimeout))
	t.Run("TTR", withQueue(testTTR))
	t.Run("Post with Pick running", withQueue(testPostAfterPick))
}

func testPickEmpty(t *testing.T, q workqueue.Queue) {
	entry, err := q.Pick(1 * time.Millisecond)

	if err != nil {
		t.Fatalf("Pick on an empty queue failed: %s", err)
	}

	if entry != nil {
		t.Errorf("Pick on empty queue returned an entry")
	}
}

func testPostPick(t *testing.T, q workqueue.Queue) {
	data := "testPostPick"
	ttr := 1 * time.Hour

	if err := q.Post(data, ttr); err != nil {
		t.Fatalf("Posting a job failed: %s", err)
	}

	if entry, err := q.Pick(1 * time.Millisecond); err != nil {
		t.Fatalf("Picking a posted job failed: %s", err)
	} else {
		if entry == nil {
			t.Errorf("Didn't pick any job")
			return
		}

		if str, ok := entry.Data.(string); !ok || str != data {
			t.Errorf("Unexpected job data, got %v, expected %v", entry.Data, data)
		}

		if entry.TTR != ttr {
			t.Errorf("Unexpected job TTR, got %d, expected %d", entry.TTR, ttr)
		}

		if err := q.Finish(entry); err != nil {
			t.Errorf("Finish returned an error")
		}
	}
}

func testPickTimeout(t *testing.T, q workqueue.Queue) {
	before := time.Now()

	entry, err := q.Pick(100 * time.Millisecond)

	after := time.Now()

	if err != nil {
		t.Errorf("Pick returned an error")
	}

	if entry != nil {
		t.Errorf("Pick return an non-nil entry")
	}

	if after.Sub(before) < 50*time.Millisecond {
		t.Errorf("Pick didn't sleep long enough")
	}

	// We need to set a long enough timeout here, since the Redis queue cannot
	// wait less than one second
	if after.Sub(before) > 5*time.Second {
		t.Errorf("Pick slept too long")
	}
}

func testTTR(t *testing.T, q workqueue.Queue) {
	data := "hello"

	if err := q.Post(data, 50*time.Millisecond); err != nil {
		t.Fatalf("Error while posting job: %s", err)
	}

	entry, err := q.Pick(1 * time.Millisecond)

	if err != nil {
		t.Fatalf("Error while picking job: %s", err)
	}

	if entry == nil {
		t.Fatalf("First pick returned a nil entry")
	}

	// Once the sleep completes, the task should have been put back to the
	// pending queue.
	time.Sleep(150 * time.Millisecond)

	entry, err = q.Pick(1 * time.Millisecond)

	if err != nil {
		t.Fatalf("Error while picking job: %s", err)
	}

	if entry == nil {
		t.Errorf("Second pick returned a nil entry")
	} else if err := q.Finish(entry); err != nil {
		t.Errorf("Finish returned an error: %s", err)
	}

	// The task has been finished, so it souldn't be rescheduled even after its
	// TTR expires
	time.Sleep(150 * time.Millisecond)

	entry, err = q.Pick(1 * time.Millisecond)

	if err != nil {
		t.Fatalf("Error while picking job: %s", err)
	}

	if entry != nil {
		t.Errorf("Third pick should have returned a nil entry")
	}
}

func testPostAfterPick(t *testing.T, q workqueue.Queue) {
	data := "hello"
	result := make(chan *workqueue.JobEntry)

	go func() {
		if value, err := q.Pick(1 * time.Second); err != nil {
			t.Errorf("Pick returned an error: %s", err)
		} else {
			result <- value
		}
	}()

	// Give the goroutine above some time to start
	time.Sleep(150 * time.Millisecond)

	if err := q.Post(data, 1*time.Second); err != nil {
		t.Errorf("Post returned an error: %s", err)
	}

	select {
	case <-time.After(1 * time.Second):
		t.Errorf("Timeout waiting for result")
	case res := <-result:
		if res == nil {
			t.Errorf("Didn't pick any job")
			return
		}

		if receivedData, ok := res.Data.(string); !ok || receivedData != data {
			t.Errorf("Unexpected Pick result, expected %v, got %v", data, receivedData)
		}

		if err := q.Finish(res); err != nil {
			t.Errorf("Finish returned an error: %s", err)
		}
	}
}
