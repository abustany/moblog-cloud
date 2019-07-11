package gitserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"

	"github.com/gorilla/mux"

	"github.com/pkg/errors"

	"github.com/abustany/moblog-cloud/pkg/adminserver"
	"github.com/abustany/moblog-cloud/pkg/middlewares"
)

const (
	urlVarUsername   = "username"
	urlVarRepository = "repository"
	validIDStringRE  = `[a-zA-Z][a-zA-Z0-9\.\-_]+`
)

type Server struct {
	baseDir            string
	templateRepository string
	router             *mux.Router
	adminClient        *adminserver.Client
	adminServerURL     *url.URL
}

type userSession struct {
	username string
	blogs    []string
}

func userSessionFromRequest(r *http.Request, adminServerURL *url.URL) (*userSession, error) {
	authCookie, err := r.Cookie(adminserver.AuthCookieName)

	if err == http.ErrNoCookie {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "Error while decoding auth cookie")
	}

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})

	if err != nil {
		return nil, errors.Wrap(err, "Error while creating cookie jar")
	}

	jar.SetCookies(adminServerURL, []*http.Cookie{authCookie})

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Jar:     jar,
	}

	adminClient, err := adminserver.NewClientWithOptions(adminServerURL.String(), httpClient)

	if err != nil {
		return nil, errors.Wrap(err, "Error while creating admin server client")
	}

	me, err := adminClient.Whoami()

	if err != nil {
		return nil, errors.Wrap(err, "Error while retrieving user information")
	}

	blogs, err := adminClient.ListBlogs()

	if err != nil {
		return nil, errors.Wrap(err, "Error while retrieving blog list")
	}

	blogNames := make([]string, len(blogs))

	for i, blog := range blogs {
		blogNames[i] = blog.Slug
	}

	return &userSession{me.Username, blogNames}, nil
}

func New(baseDir, templateRepository, adminServerURL string) (*Server, error) {
	adminClient, err := adminserver.NewClient(adminServerURL)

	if err != nil {
		return nil, errors.Wrap(err, "Error while creating admin server client")
	}

	adminServerURLParsed, err := url.Parse(adminServerURL)

	if err != nil {
		return nil, errors.Wrap(err, "Error while parsing admin server URL")
	}

	s := Server{
		baseDir:            baseDir,
		templateRepository: templateRepository,
		router:             mux.NewRouter(),
		adminClient:        adminClient,
		adminServerURL:     adminServerURLParsed,
	}

	repoRouter := s.router.PathPrefix("/{" + urlVarUsername + ":" + validIDStringRE + "}/{" + urlVarRepository + ":" + validIDStringRE + "}").Subrouter()

	repoRouter.
		Methods("GET").
		Path("/info/refs").
		Handler(middlewares.WithLogging(s.withValidRepository(http.HandlerFunc(s.serveInfoRefsHTTP))))

	repoRouter.
		Methods("POST").
		Path("/git-upload-pack").
		Handler(middlewares.WithLogging(s.withValidRepository(http.HandlerFunc(s.serveUploadPackHTTP))))

	repoRouter.
		Methods("POST").
		Path("/git-receive-pack").
		Handler(middlewares.WithLogging(s.withValidRepository(http.HandlerFunc(s.serveReceivePackHTTP))))

	s.router.
		PathPrefix("/").
		Handler(middlewares.WithLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, "Not found")
		})))

	return &s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) ensureRepository(ctx context.Context, session *userSession, repository string) error {
	repoPath := s.repositoryPath(session.username, repository)

	exists, err := isDir(repoPath)

	if err != nil {
		return errors.Wrap(err, "Error while checking if repository directory exists")
	}

	if exists {
		return nil
	}

	if err := os.MkdirAll(repoPath, 0700); err != nil {
		return errors.Wrap(err, "Error while creating repository directory")
	}

	if stderr, err := runGit(ctx, nil, nil, "clone", "--bare", s.templateRepository, repoPath); err != nil {
		return errors.Wrapf(err, "Error while cloning template repository (stderr: %s)", stderr)
	}

	return nil
}

func (s *Server) withValidRepository(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, repository := getRequestUsernameRepository(r)
		session, err := userSessionFromRequest(r, s.adminServerURL)

		if err != nil {
			log.Printf("Error while handling session: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if session == nil || session.username != username {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if err := s.ensureRepository(r.Context(), session, repository); err != nil {
			log.Printf("Error while ensuring that repository %s/%s exists: %s", username, repository, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

func removeGitRepositorySuffix(repository string) string {
	const gitSuffix = ".git"

	if strings.HasSuffix(repository, gitSuffix) {
		return repository[0 : len(repository)-len(gitSuffix)]
	}

	return repository
}

func (s *Server) repositoryPath(username, repository string) string {
	return path.Join(s.baseDir, username, removeGitRepositorySuffix(repository))
}

func writeGitPacket(w io.Writer, s string) error {
	if len(s) > 65535-4 {
		panic("Packet too big")
	}

	_, err := fmt.Fprintf(w, "%04x%s", int16(len(s)+4), s)

	return err
}

func isDir(path string) (bool, error) {
	s, err := os.Stat(path)

	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return s.IsDir(), nil
}

func getRequestUsernameRepository(r *http.Request) (string, string) {
	vars := mux.Vars(r)
	return vars[urlVarUsername], vars[urlVarRepository]
}

func runGit(ctx context.Context, stdin io.Reader, stdout io.Writer, args ...string) (stderr string, err error) {
	gitPath, err := exec.LookPath("git")

	if err != nil {
		return "", errors.Wrap(err, "Cannot find git in PATH")
	}

	var stderrBuffer bytes.Buffer
	gitCmd := exec.CommandContext(ctx, gitPath, args...)
	gitCmd.Env = []string{"GIT_TERMINAL_PROMPT=0"}
	gitCmd.Stdin = stdin
	gitCmd.Stdout = stdout
	gitCmd.Stderr = &stderrBuffer

	log.Printf("Running git %v", args)

	err = gitCmd.Run()
	stderr = strings.TrimSpace(stderrBuffer.String())

	return
}

func runGitService(ctx context.Context, stdin io.Reader, stdout io.Writer, repoPath, serviceName string, extraArgs ...string) (stderr string, err error) {
	args := []string{serviceName, "--stateless-rpc"}
	args = append(args, extraArgs...)
	args = append(args, repoPath)

	return runGit(ctx, stdin, stdout, args...)
}

func (s *Server) serveInfoRefsHTTP(w http.ResponseWriter, r *http.Request) {
	serviceName := r.URL.Query().Get("service")

	if serviceName != "git-upload-pack" && serviceName != "git-receive-pack" {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "Unknown service")
		return
	}

	serviceName = serviceName[len("git-"):]

	username, repository := getRequestUsernameRepository(r)
	repoPath := s.repositoryPath(username, repository)

	w.Header().Set("Content-Type", "application/x-git-"+serviceName+"-advertisement")
	w.Header().Set("Cache-Control", "no-cache")

	writeGitPacket(w, "# service=git-"+serviceName+"\n")
	io.WriteString(w, "0000")

	stderr, err := runGitService(r.Context(), nil, w, repoPath, serviceName, "--advertise-refs")

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Error while getting refs for %s/%s: %s (stderr: %s)", username, repository, err, stderr)
		return
	}
}

func clientAccepts(header http.Header, format string) bool {
	for _, x := range header["Accept"] {
		if format == x {
			return true
		}
	}

	return false
}

func (s *Server) serveGitServiceHTTP(w http.ResponseWriter, r *http.Request, serviceName string) {
	if r.Header.Get("Content-Type") != "application/x-git-"+serviceName+"-request" || !clientAccepts(r.Header, "application/x-git-"+serviceName+"-result") {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "Invalid formats")
		return
	}

	username, repository := getRequestUsernameRepository(r)
	repoPath := s.repositoryPath(username, repository)

	w.Header().Set("Content-Type", "application/x-git-"+serviceName+"-result")
	w.Header().Set("Cache-Control", "no-cache")

	stderr, err := runGitService(r.Context(), r.Body, w, repoPath, serviceName)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Error while doing %s for %s/%s: %s (stderr: %s)", serviceName, username, repository, err, stderr)
		return
	}
}

func (s *Server) serveUploadPackHTTP(w http.ResponseWriter, r *http.Request) {
	s.serveGitServiceHTTP(w, r, "upload-pack")
}

func (s *Server) serveReceivePackHTTP(w http.ResponseWriter, r *http.Request) {
	s.serveGitServiceHTTP(w, r, "receive-pack")
}
