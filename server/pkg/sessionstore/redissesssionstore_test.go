package sessionstore_test

import (
	"os"
	"testing"

	"github.com/abustany/moblog-cloud/pkg/sessionstore"
)

func TestRedisSessionStore(t *testing.T) {
	redisURL := os.Getenv("REDIS_URL")

	if redisURL == "" {
		t.Skip("REDIS_URL environment variable not defined")
	}

	store, err := sessionstore.NewRedisSessionStore(redisURL)

	if err != nil {
		t.Fatalf("Error while creating session store: %s", err)
	}

	testSessionStore(t, store)
}
