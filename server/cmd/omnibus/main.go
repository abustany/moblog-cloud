package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	assetfs "github.com/elazarl/go-bindata-assetfs"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"

	_ "github.com/lib/pq"

	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/s3blob"

	adminui "github.com/abustany/moblog-cloud/omnibus-adminui"
	"github.com/abustany/moblog-cloud/pkg/adminserver"
	"github.com/abustany/moblog-cloud/pkg/gitserver"
	"github.com/abustany/moblog-cloud/pkg/sessionstore"
	"github.com/abustany/moblog-cloud/pkg/userstore"
	"github.com/abustany/moblog-cloud/pkg/worker"
	"github.com/abustany/moblog-cloud/pkg/workqueue"
)

const AdminUiPrefix = "admin"
const FileUriPrefix = "file://"

func parseKey(key, usage string, length int) []byte {
	if len(key) == 0 {
		log.Fatalf("Cookie %s key is required", usage)
	}

	decoded, err := hex.DecodeString(key)

	if err != nil {
		log.Fatalf("Error while decoding %s key: %s", usage, err)
	}

	if len(decoded) != length {
		log.Fatalf("Invalid length for %s key: expected %d bytes, got %d", usage, length, len(key))
	}

	return decoded
}

func main() {
	listenAddress := flag.String("listen", "127.0.0.1:8080", "Address to listen on, of the form IP:PORT")
	dbURL := flag.String("db", "", "URL to the PostgreSQL server. If not set, the DB_URL environment variable is used. If the value is \"memory\", a non-persistent, in-memory user store is used.")
	cookieSignKeyString := flag.String("cookieSignKey", "", "Key used to sign cookies sent to users (64 hex encoded bytes). Auto generated if left empty.")
	cookieCryptKeyString := flag.String("cookieCryptKey", "", "Key used to encrypt cookies sent to users (32 hex encoded bytes). Auto generated if left empty")
	redisSessionURL := flag.String("redisSessionURL", "", "Redis URL of the server to use for storing sessions (if not specified, sessions are kept in memory only)")
	repositoryBase := flag.String("repositoryBase", "", "Base path where user repositories are stored")
	workDir := flag.String("workDir", "", "Directory where to checkout the blog source and do the rendering work")
	themeRepositoryURL := flag.String("themeRepository", "", "URL of the Git repository holding the blog theme")
	blogOutputURL := flag.String("blogOutput", "", "Where to store the generated blog files. See https://gocloud.dev/howto/blob/ for supported URLs.")

	flag.Parse()

	cookieSignKey := parseKey(*cookieSignKeyString, "signing", 64)
	cookieCryptKey := parseKey(*cookieCryptKeyString, "encryption", 32)
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

	if *redisSessionURL == "" {
		log.Fatalf("Missing option: -redisSessionURL")
	}

	sessionStore, err := sessionstore.NewRedisSessionStore(*redisSessionURL)

	if err != nil {
		log.Fatalf("Error while creating session store: %s", err)
	}

	jobQueue, err := workqueue.NewMemoryQueue()

	if err != nil {
		log.Fatalf("Error while creating job queue: %s", err)
	}

	adminServer, err := adminserver.New("/api", secureCookie, userStore, sessionStore)

	if err != nil {
		log.Fatalf("Error while creating adminserver: %s", err)
	}

	host, port, err := net.SplitHostPort(*listenAddress)

	if err != nil {
		log.Fatalf("Could not parse port from listen address %s", *listenAddress)
	}

	if host == "0.0.0.0" {
		host = "127.0.0.1"
	}

	adminServerURL := fmt.Sprintf("http://%s:%s/api", host, port)
	gitServerURL := fmt.Sprintf("http://%s:%s/git", host, port)

	if *repositoryBase == "" {
		log.Fatalf("Missing option: -repositoryBase")
	}

	gitServer, err := gitserver.New("/git", *repositoryBase, adminServerURL, jobQueue)

	if err != nil {
		log.Fatalf("Error while creating gitserver: %s", err)
	}

	if *workDir == "" {
		log.Fatalf("Missing option: -workDir")
	}

	if *themeRepositoryURL == "" {
		log.Fatalf("Missing option: -themeRepository")
	}

	if *blogOutputURL == "" {
		log.Fatalf("Missing option: -blogOutput")
	}

	worker, err := worker.New(jobQueue, adminServerURL, gitServerURL, *workDir, *themeRepositoryURL, *blogOutputURL)

	if err != nil {
		log.Fatalf("Error while initializing worker: %s", err)
	}

	defer worker.Stop()

	router := mux.NewRouter()
	router.PathPrefix("/api").Handler(adminServer)
	router.PathPrefix("/git").Handler(gitServer)
	router.PathPrefix("/" + AdminUiPrefix).Handler(makeAdminFileServer())

	if strings.HasPrefix(*blogOutputURL, FileUriPrefix) {
		blogDirectory := (*blogOutputURL)[len(FileUriPrefix):]
		log.Printf("Local blog output detected, activating file server on / for %s", blogDirectory)
		router.PathPrefix("/").Handler(http.FileServer(http.Dir(blogDirectory)))
	}

	log.Printf("Listening on %s", *listenAddress)
	err = http.ListenAndServe(*listenAddress, router)

	log.Fatalf("Error listening on %s: %s", *listenAddress, err)
}

func stripAdminPrefix(path string) string {
	path = path[len(AdminUiPrefix):]

	for len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	return path
}

func makeAdminFileServer() http.Handler {
	return http.FileServer(&assetfs.AssetFS{
		Asset: func(path string) ([]byte, error) {
			path = stripAdminPrefix(path)
			log.Printf("Asset request: %s", path)
			return adminui.Asset(path)
		},
		AssetDir: func(path string) ([]string, error) {
			path = stripAdminPrefix(path)
			log.Printf("AssetDir request: %s", path)
			return adminui.AssetDir(path)
		},
	})
}
