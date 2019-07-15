package workqueue

import (
	"time"

	"github.com/pkg/errors"
)

type JobEntry struct {
	ID   string
	TTR  time.Duration
	Data interface{}
}

type Queue interface {
	Post(job interface{}, ttr time.Duration) error

	// Gets a job from the queue and reserves it
	Pick(timeout time.Duration) (*JobEntry, error)
	Finish(entry *JobEntry) error
}

var ErrQueueFull = errors.New("Queue is full")
