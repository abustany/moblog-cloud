package gitserver_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"golang.org/x/net/publicsuffix"

	"github.com/abustany/moblog-cloud/pkg/adminserver"
	"github.com/abustany/moblog-cloud/pkg/gitserver"
	"github.com/abustany/moblog-cloud/pkg/testutils"
	"github.com/abustany/moblog-cloud/pkg/userstore"
)

func makeJarWithCookie(t *testing.T, serverURL string, cookie *http.Cookie) http.CookieJar {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})

	if err != nil {
		t.Fatalf("Error while creating cookie jar: %s", err)
	}

	parsedURL, err := url.Parse(serverURL)

	if err != nil {
		t.Fatalf("Error while parsing server URL: %s", err)
	}

	jar.SetCookies(parsedURL, []*http.Cookie{cookie})

	return jar
}

type Context struct {
	gitServerURL   string
	httpClient     *http.Client
	adminClient    *adminserver.Client
	username       string
	workDir        string
	authCookieFile string
}

func TestGitService(t *testing.T) {
	testutils.FlushDB(t)

	adminServer := httptest.NewServer(testutils.NewAdminServer(t))
	defer adminServer.Close()

	repositoriesDir, err := ioutil.TempDir("", "gitserver-repositories")

	if err != nil {
		t.Fatalf("Error while creating git directory: %s", err)
	}

	defer os.RemoveAll(repositoriesDir)

	const templateRepoDir = "testdata/repo-template"

	gitServerHandler, err := gitserver.New(repositoriesDir, templateRepoDir, adminServer.URL)

	if err != nil {
		t.Fatalf("Error while creating git server: %s", err)
	}

	gitServer := httptest.NewServer(gitServerHandler)
	defer gitServer.Close()

	adminClient, err := adminserver.NewClient(adminServer.URL)

	if err != nil {
		t.Fatalf("Error while creating admin RPC client: %s", err)
	}

	user := userstore.User{
		Username: "gituser",
		Password: "gitster",
	}

	if err := adminClient.CreateUser(user); err != nil {
		t.Fatalf("Error while creating user: %s", err)
	}

	if err := adminClient.Login(user.Username, user.Password); err != nil {
		t.Fatalf("Error while logging in: %s", err)
	}

	authCookie, err := adminClient.AuthCookie()

	if err != nil {
		t.Fatalf("Error while retrieving auth cookie from admin client: %s", err)
	}

	httpClient := &http.Client{
		Jar: makeJarWithCookie(t, adminServer.URL, authCookie),
	}

	workDir, err := ioutil.TempDir("", "gitserver-workdir")

	if err != nil {
		t.Fatalf("Error while creating work directory: %s", err)
	}

	// Store the auth cookie as a text file so that git can use it later

	authCookieFile := path.Join(workDir, "auth_cookie.txt")

	if err := ioutil.WriteFile(authCookieFile, []byte("Set-Cookie: "+authCookie.String()+"\n"), 0600); err != nil {
		t.Fatalf("Error while writing auth cookie to file: %s", err)
	}

	withContext := func(f func(*testing.T, Context)) func(*testing.T) {
		return func(t *testing.T) {
			ctx := Context{
				gitServerURL:   gitServer.URL,
				httpClient:     httpClient,
				adminClient:    adminClient,
				username:       user.Username,
				workDir:        workDir,
				authCookieFile: authCookieFile,
			}

			f(t, ctx)
		}
	}

	t.Run("Authentication", withContext(testAuthentication))
	t.Run("Clone", withContext(testClone))
	t.Run("Push", withContext(testPush))
}

func testAuthentication(t *testing.T, ctx Context) {
	infoRefsURL := ctx.gitServerURL + "/gituser/reponame/info/refs?service=git-upload-pack"

	res, err := ctx.httpClient.Get(infoRefsURL)

	if err != nil {
		t.Fatalf("Error while sending info/refs request: %s", err)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Unexpected HTTP status code for a non-existing repository: %d (expected 200)", res.StatusCode)
	}

	newClient := &http.Client{}
	res, err = newClient.Get(infoRefsURL)

	if err != nil {
		t.Fatalf("Error while sending info/refs request: %s", err)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("Unexpected HTTP status code for a request without authentication: %d (expected 401)", res.StatusCode)
	}
}

func git(t *testing.T, args ...string) {
	gitPath, err := exec.LookPath("git")

	if err != nil {
		t.Fatalf("Cannot find git in PATH")
	}

	var stderrBuffer bytes.Buffer
	gitCmd := exec.Command(gitPath, args...)
	gitCmd.Env = []string{"GIT_TERMINAL_PROMPT=0"}
	gitCmd.Stdin = nil
	gitCmd.Stdout = nil
	gitCmd.Stderr = &stderrBuffer

	t.Logf("Running git %v", args)

	if err := gitCmd.Run(); err != nil {
		t.Fatalf("Git command failed. Stderr: %s", stderrBuffer.String())
	}
}

func testClone(t *testing.T, ctx Context) {
	blog := userstore.Blog{
		Slug:        "my-blog",
		DisplayName: "my fancy blog",
	}

	if err := ctx.adminClient.CreateBlog(blog); err != nil {
		t.Fatalf("Error while creating blog: %s", err)
	}

	blogURL := ctx.gitServerURL + "/" + ctx.username + "/" + blog.Slug
	blogPath := path.Join(ctx.workDir, blog.Slug)

	git(t, "-c", "http.cookieFile="+ctx.authCookieFile, "clone", blogURL, blogPath)

	if readme, err := ioutil.ReadFile(path.Join(blogPath, "README")); err != nil {
		t.Errorf("Error while reading README: %s", err)
	} else {
		if !strings.Contains(string(readme), "Welcome to your new blog!") {
			t.Errorf("Unexpected README contents: %s", string(readme))
		}
	}
}

func testPush(t *testing.T, ctx Context) {
	blogPath := path.Join(ctx.workDir, "my-blog")

	if err := ioutil.WriteFile(path.Join(blogPath, "README"), []byte("Changed by myself"), 0600); err != nil {
		t.Fatalf("Error while writing README: %s", err)
	}

	git(t, "-C", blogPath, "-c", "user.name=Tester", "-c", "user.email=tester@qa.org", "commit", "-m", "Change the README", "README")
	git(t, "-C", blogPath, "-c", "http.cookieFile="+ctx.authCookieFile, "push", "origin", "master")
}
