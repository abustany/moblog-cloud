package testutils

import (
	"os"
	"testing"

	"github.com/gorilla/securecookie"

	"github.com/abustany/moblog-cloud/pkg/adminserver"
	"github.com/abustany/moblog-cloud/pkg/sessionstore"
	"github.com/abustany/moblog-cloud/pkg/userstore"
)

func generateSecureCookie(t *testing.T) *securecookie.SecureCookie {
	signKey := securecookie.GenerateRandomKey(64)
	encryptKey := securecookie.GenerateRandomKey(32)

	if signKey == nil || encryptKey == nil {
		t.Fatalf("Error while generating cookie signing keys")
	}

	return securecookie.New(signKey, encryptKey)
}

const DBURLEnvVar = "DB_URL"

func NewAdminServer(t *testing.T) *adminserver.Server {
	dbURL := os.Getenv(DBURLEnvVar)

	var userStore userstore.UserStore
	var err error

	if dbURL == "" {
		t.Logf("Using an in-memory user store")
		userStore, err = userstore.NewMemoryUserStore()
	} else {
		t.Logf("Using PostgreSQL at %s for the user store", dbURL)
		userStore, err = userstore.NewSQLUserStore("postgres", dbURL)
	}

	if err != nil {
		t.Fatalf("Error while creating user store: %s", err)
	}

	sessionStore, err := sessionstore.NewMemorySessionStore()

	if err != nil {
		t.Fatalf("Error while creating session store")
	}

	return adminserver.New(generateSecureCookie(t), userStore, sessionStore)
}
