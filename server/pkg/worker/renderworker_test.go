package worker_test

import (
	"io/ioutil"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/pkg/errors"

	_ "gocloud.dev/blob/fileblob"

	"github.com/abustany/moblog-cloud/pkg/adminserver"
	"github.com/abustany/moblog-cloud/pkg/gitserver"
	"github.com/abustany/moblog-cloud/pkg/testutils"
	"github.com/abustany/moblog-cloud/pkg/userstore"
	"github.com/abustany/moblog-cloud/pkg/worker"
	"github.com/abustany/moblog-cloud/pkg/workqueue"
)

func TestRenderBlog(t *testing.T) {
	testutils.FlushDB(t)

	adminServer := httptest.NewServer(testutils.NewAdminServer(t))
	defer adminServer.Close()

	queue, err := workqueue.NewMemoryQueue()

	if err != nil {
		t.Fatalf("Error while creating work queue: %s", err)
	}

	defer queue.Stop()

	repositoriesDir := testutils.TempDir(t, "gitserver-repositories")
	defer os.RemoveAll(repositoriesDir)

	gitServerHandler, err := gitserver.New(repositoriesDir, adminServer.URL, queue)

	if err != nil {
		t.Fatalf("Error while creating git server: %s", err)
	}

	gitServer := httptest.NewServer(gitServerHandler)
	defer gitServer.Close()

	workDir := testutils.TempDir(t, "worker-workdir")
	defer os.RemoveAll(workDir)

	themesDirectory, err := filepath.Abs(path.Join("testdata", "blog-theme"))

	if err != nil {
		t.Fatalf("Error while getting blog theme path: %s", err)
	}

	destDir := testutils.TempDir(t, "worker-destdir")
	defer os.RemoveAll(destDir)

	w, err := worker.New(queue, adminServer.URL, gitServer.URL, workDir, "file://"+themesDirectory, "file://"+destDir)

	if err != nil {
		t.Fatalf("Error creating worker: %s", err)
	}

	defer w.Stop()

	adminClient, err := adminserver.NewClient(adminServer.URL)

	user := userstore.User{
		Username: "renderer",
		Password: "don't tell",
	}

	blog := userstore.Blog{
		Slug:        "myblog",
		DisplayName: "My fancy blog",
	}

	if err := adminClient.CreateUser(user); err != nil {
		t.Fatalf("Error while creating test user: %s", err)
	}

	if err := adminClient.Login(user.Username, user.Password); err != nil {
		t.Fatalf("Error while logging in: %s", err)
	}

	if err := adminClient.CreateBlog(blog); err != nil {
		t.Fatalf("Error while creating blog: %s", err)
	}

	// The part below would be done by a real client: init an empty directory,
	// initialize the content, push it back
	userDir := testutils.TempDir(t, "user")
	defer os.RemoveAll(userDir)

	authCookieFile := path.Join(userDir, "auth_cookie.txt")
	testutils.SaveAdminClientAuthCookieToFile(t, adminClient, authCookieFile)

	blogURL := gitServer.URL + "/" + user.Username + "/" + blog.Slug
	blogDirectory := path.Join(userDir, blog.Slug)
	postsDirectory := path.Join(blogDirectory, "content", "post")
	const postMardown = `
+++
date = "2019-07-19T15:26:56+01:00"
description = ""
draft = false
position = "52.48, 13.44"
title = "My first post"
+++

Hello world!
`

	if err := os.MkdirAll(postsDirectory, 0700); err != nil {
		t.Fatalf("Error while creating blog directory: %s", err)
	}

	if err := ioutil.WriteFile(path.Join(postsDirectory, "first.md"), []byte(postMardown), 0600); err != nil {
		t.Fatalf("Error while creating post file: %s", err)
	}

	testutils.Git(t, "init", blogDirectory)
	testutils.Git(t, "-C", blogDirectory, "add", ".")
	testutils.Git(t, "-C", blogDirectory, "-c", "user.name=Renderer", "-c", "user.email=renderer@qa.org", "commit", "-m", "Commit first post")
	testutils.Git(t, "-c", "http.cookieFile="+authCookieFile, "-C", blogDirectory, "push", blogURL, "master")

	indexHTMLPath := path.Join(destDir, user.Username, blog.Slug, "index.html")

	if err := waitForFile(indexHTMLPath, 3*time.Second); err != nil {
		t.Fatalf("Error while waiting for %s to be produced: %s", indexHTMLPath, err)
	}
}

func waitForFile(filename string, timeout time.Duration) error {
	start := time.Now()

	for time.Now().Sub(start) < timeout {
		_, err := os.Stat(filename)

		if os.IsNotExist(err) {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if err != nil {
			return errors.Wrap(err, "Error while waiting for file")
		}

		return nil
	}

	return errors.New("timeout")
}
