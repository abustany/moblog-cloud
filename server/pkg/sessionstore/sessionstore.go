package sessionstore

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type Session struct {
	Sid      string
	Expires  time.Time
	Username string
}

type SessionStore interface {
	Set(session Session) error
	Get(sid string) (*Session, error)
	Delete(sid string) error
}

func GenerateSessionID() string {
	return uuid.NewV4().String()
}
