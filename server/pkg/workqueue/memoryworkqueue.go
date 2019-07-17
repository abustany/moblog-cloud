package workqueue

import (
	"sync"
	"time"

	"github.com/abustany/moblog-cloud/pkg/idgenerator"
)

const MemoryMaxPendingJobs = 1000

type memoryEntryMetadata struct {
	entry   *JobEntry
	started time.Time
}

type MemoryQueue struct {
	idGenerator *idgenerator.StringIdGenerator
	pendingChan chan *JobEntry
	metadata    map[string]memoryEntryMetadata
	mutex       sync.Mutex // protects reserved and metadata
	groomTicker *time.Ticker
}

func NewMemoryQueue() (*MemoryQueue, error) {
	q := &MemoryQueue{
		idGenerator: &idgenerator.StringIdGenerator{},
		pendingChan: make(chan *JobEntry, MemoryMaxPendingJobs),
		groomTicker: time.NewTicker(GroomInterval),
		metadata:    map[string]memoryEntryMetadata{},
	}

	go func() {
		for range q.groomTicker.C {
			q.groom()
		}
	}()

	return q, nil
}

func (q *MemoryQueue) Stop() {
	q.groomTicker.Stop()
}

func (q *MemoryQueue) groom() {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	now := time.Now()

	for _, m := range q.metadata {
		if now.After(m.started.Add(m.entry.TTR)) {
			// Job took too long to run, put it back in the ready queue
			delete(q.metadata, m.entry.ID)
			q.pendingChan <- m.entry
		}
	}
}

func (q *MemoryQueue) Post(job interface{}, ttr time.Duration) error {
	entry := &JobEntry{
		ID:   q.idGenerator.Next(),
		TTR:  ttr,
		Data: job,
	}

	select {
	case q.pendingChan <- entry:
		return nil
	default:
		return ErrQueueFull
	}
}

func (q *MemoryQueue) reserveEntry(entry *JobEntry) {
	q.mutex.Lock()
	q.metadata[entry.ID] = memoryEntryMetadata{entry: entry, started: time.Now()}
	q.mutex.Unlock()
}

func (q *MemoryQueue) Pick(timeout time.Duration) (*JobEntry, error) {
	t := time.NewTimer(timeout)
	defer t.Stop()

	select {
	case entry := <-q.pendingChan:
		q.reserveEntry(entry)
		return entry, nil
	case <-t.C:
		return nil, nil
	}
}

func (q *MemoryQueue) Finish(entry *JobEntry) error {
	q.mutex.Lock()
	delete(q.metadata, entry.ID)
	q.mutex.Unlock()

	return nil
}
