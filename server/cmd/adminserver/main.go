package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/securecookie"

	_ "github.com/lib/pq"

	"github.com/abustany/moblog-cloud/pkg/adminserver"
	"github.com/abustany/moblog-cloud/pkg/sessionstore"
	"github.com/abustany/moblog-cloud/pkg/userstore"
)

var listenAddress = flag.String("listen", "127.0.0.1:8080", "Address to listen on, of the form IP:PORT")
var dbURL = flag.String("db", "", "URL to the PostgreSQL server. If not set, the DB_URL environment variable is used.")
var cookieSignKeyString = flag.String("cookieSignKey", "", "Key used to sign cookies sent to users (64 hex encoded bytes). Auto generated if left empty.")
var cookieCryptKeyString = flag.String("cookieCryptKey", "", "Key used to encrypt cookies sent to users (32 hex encoded bytes). Auto generated if left empty")

func main() {
	flag.Parse()

	cookieSignKey := ensureKey(*cookieSignKeyString, "signing", 64)
	cookieCryptKey := ensureKey(*cookieCryptKeyString, "encryption", 32)
	secureCookie := securecookie.New(cookieSignKey, cookieCryptKey)

	if *dbURL == "" {
		*dbURL = os.Getenv("DB_URL")
	}

	if *dbURL == "" {
		log.Fatal("No database URL set. Use -db or the DB_URL environment variable.")
	}

	userStore, err := userstore.NewSQLUserStore("postgres", *dbURL)

	if err != nil {
		log.Fatalf("Error while creating user store: %s", err)
	}

	sessionStore, _ := sessionstore.NewMemorySessionStore()

	s := adminserver.New(secureCookie, userStore, sessionStore)

	log.Printf("Listening on %s", *listenAddress)
	err = http.ListenAndServe(*listenAddress, s)

	log.Fatalf("Error listening on %s: %s", *listenAddress, err)
}

func ensureKey(keyStr, usage string, length int) []byte {
	var key []byte
	var err error

	if len(keyStr) > 0 {
		key, err = hex.DecodeString(keyStr)

		if err != nil {
			log.Fatalf("Error while decoding %s key: %s", usage, err)
		}
	} else {
		key = make([]byte, length)

		if _, err := rand.Read(key); err != nil {
			log.Fatalf("Error while generating %s key: %s", usage, err)
		}
	}

	if len(key) != length {
		log.Fatalf("Invalid length for %s key: expected %d bytes, got %d", usage, length, len(key))
	}

	return key
}
