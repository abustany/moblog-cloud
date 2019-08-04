package sessionstore_test

import (
	"testing"
	"time"

	"github.com/abustany/moblog-cloud/pkg/sessionstore"
)

func testSessionStore(t *testing.T, s sessionstore.SessionStore) {
	withStore := func(f func(*testing.T, sessionstore.SessionStore)) func(*testing.T) {
		return func(t *testing.T) {
			f(t, s)
		}
	}

	t.Run("Set Get Delete", withStore(testSetGetDelete))
}

func testSetGetDelete(t *testing.T, store sessionstore.SessionStore) {
	session := sessionstore.Session{
		Sid:      "session",
		Expires:  time.Now().Add(time.Hour),
		Username: "user",
	}

	if s, err := store.Get(session.Sid); err != nil {
		t.Errorf("Error while retrieving a non-existing session: %s", err)
	} else {
		if s != nil {
			t.Errorf("Get on a non-existing session id should return nil")
		}
	}

	if err := store.Set(session); err != nil {
		t.Errorf("Error while saving a session: %s", err)
	}

	checkSession := func(expected sessionstore.Session) {
		if s, err := store.Get(session.Sid); err != nil {
			t.Errorf("Error while retrieving session %s: %s", session.Sid, err)
		} else {
			if s == nil {
				t.Errorf("Get on an existing session returned nil")
			} else {
				if *s != expected {
					t.Errorf("Retrieving session doesn't match, got %v, expected %v", *s, expected)
				}
			}
		}
	}

	checkSession(session)

	session.Expires = time.Now().Add(2 * time.Hour)

	if err := store.Set(session); err != nil {
		t.Errorf("Error while overwriting a session: %s", err)
	}

	checkSession(session)

	if err := store.Delete(session.Sid + "does not exist"); err != nil {
		t.Errorf("Error while deleting a non existing session: %s", err)
	}

	// Check that delete did not delete our session
	checkSession(session)

	if err := store.Delete(session.Sid); err != nil {
		t.Errorf("Error while deleting session: %s", err)
	}

	if s, err := store.Get(session.Sid); err != nil {
		t.Errorf("Error while retrieving session after deletion: %s", err)
	} else {
		if s != nil {
			t.Errorf("Get after delete should return a nil session")
		}
	}
}
