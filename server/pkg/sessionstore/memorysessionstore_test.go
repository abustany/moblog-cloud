package sessionstore_test

import (
	"testing"

	"github.com/abustany/moblog-cloud/pkg/sessionstore"
)

func TestMemorySessionStore(t *testing.T) {
	store, err := sessionstore.NewMemorySessionStore()

	if err != nil {
		t.Fatalf("Error while creating session store: %s", err)
	}

	testSessionStore(t, store)
}
