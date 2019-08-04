package sessionstore

import (
	"sync"
	"time"
)

type MemorySessionStore struct {
	sync.Mutex
	sessions map[string]Session
}

func NewMemorySessionStore() (*MemorySessionStore, error) {
	return &MemorySessionStore{
		sessions: make(map[string]Session),
	}, nil
}

func (s *MemorySessionStore) Set(session Session) error {
	s.Lock()
	defer s.Unlock()

	if session.Sid == "" {
		panic("Empty session ID")
	}

	s.sessions[session.Sid] = session
	return nil
}

func (s *MemorySessionStore) Get(sid string) (*Session, error) {
	s.Lock()
	defer s.Unlock()

	session := s.sessions[sid]

	if time.Now().After(session.Expires) {
		delete(s.sessions, sid)
		return nil, nil
	}

	return &session, nil
}

func (s *MemorySessionStore) Delete(sid string) error {
	s.Lock()
	defer s.Unlock()

	delete(s.sessions, sid)

	return nil
}
