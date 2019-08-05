package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"gocloud.dev/blob"

	"github.com/abustany/moblog-cloud/pkg/adminserver"
	"github.com/abustany/moblog-cloud/pkg/jobs"
	"github.com/abustany/moblog-cloud/pkg/netscapecookies"
	"github.com/abustany/moblog-cloud/pkg/userstore"
)

const blogDirectory = "blog"
const themeDirectory = "theme"
const resultDirectory = "html"

func (w *Worker) renderBlog(ctx context.Context, job *jobs.RenderJob) error {
	adminClient, err := adminserver.NewClient(w.adminServerURL)

	if err != nil {
		return errors.Wrap(err, "Error while creating adminserver client")
	}

	if err := adminClient.SetAuthCookie(&job.AuthCookie); err != nil {
		return errors.Wrap(err, "Error while setting auth cookie on client")
	}

	if err := adminClient.RefreshSession(); err != nil {
		return errors.Wrap(err, "Error while refreshing client session")
	}

	blog, err := adminClient.GetUserBlog(job.Username, job.Repository)

	if err != nil {
		return errors.Wrap(err, "Error while fetching blog information")
	}

	authCookieFile := path.Join(w.workDir, "auth_cookie.txt")

	if err := w.writeAuthCookieFile(adminClient, authCookieFile); err != nil {
		return errors.Wrap(err, "Error while writing auth cookie file")
	}

	if err := w.cloneBlog(ctx, job, authCookieFile); err != nil {
		return errors.Wrap(err, "Error while cloning blog")
	}

	if err := w.cloneTheme(ctx); err != nil {
		return errors.Wrap(err, "Error while cloning theme")
	}

	configFilePath := path.Join(w.workDir, "config.json")

	if err := w.generateConfigFile(configFilePath, blog); err != nil {
		return errors.Wrap(err, "Error while generating config file")
	}

	if err := w.runHugo(ctx, configFilePath); err != nil {
		return errors.Wrap(err, "Error while running Hugo")
	}

	if err := w.uploadHTMLFiles(ctx, job.Username, blog.Slug); err != nil {
		return errors.Wrap(err, "Error while uploading generated files")
	}

	return nil
}

func (w *Worker) writeAuthCookieFile(adminClient *adminserver.Client, authCookieFile string) error {
	authCookie, err := adminClient.AuthCookie()

	if err != nil {
		return errors.Wrap(err, "Error while retrieving auth cookie from admin client")
	}

	if authCookie == nil {
		return errors.New("Admin client has no auth cookie")
	}

	buffer := bytes.Buffer{}

	if err := netscapecookies.WriteCookie(&buffer, authCookie); err != nil {
		return errors.Wrap(err, "Error while serializing admin server cookie")
	}

	// Serialize another copy of this cookie, but for the git server

	gitAuthCookie := *authCookie
	parsedGitServerURL, err := url.Parse(w.gitServerURL)

	if err != nil {
		return errors.Wrap(err, "Error while parsing Git server URL")
	}

	gitAuthCookie.Domain = parsedGitServerURL.Host

	if err := netscapecookies.WriteCookie(&buffer, &gitAuthCookie); err != nil {
		return errors.Wrap(err, "Error while serializing git server cookie")
	}

	if err := ioutil.WriteFile(authCookieFile, buffer.Bytes(), 0600); err != nil {
		return errors.Wrap(err, "Error while writing auth cookie file")
	}

	return nil
}

func (w *Worker) cloneBlog(ctx context.Context, job *jobs.RenderJob, authCookieFile string) error {
	repoURL := w.gitServerURL + "/" + job.Username + "/" + job.Repository
	repoPath := path.Join(w.workDir, blogDirectory)

	if err := os.RemoveAll(repoPath); err != nil {
		return errors.Wrap(err, "Error while cleaning blog directory")
	}

	err := runGit(ctx, nil, "-c", "http.cookieFile="+authCookieFile, "clone", "--depth", "1", repoURL, repoPath)

	if err != nil {
		return errors.Wrap(err, "Error while cloning")
	}

	return nil
}

func (w *Worker) cloneTheme(ctx context.Context) error {
	themePath := path.Join(w.workDir, themeDirectory)

	stat, err := os.Stat(themePath)

	if os.IsNotExist(err) {
		err := runGit(ctx, nil, "clone", w.themeRepositoryURL, themePath)

		return errors.Wrap(err, "Error while cloning theme")
	}

	if err != nil {
		return errors.Wrap(err, "Error while checking if theme directory exists")
	}

	if !stat.IsDir() {
		return errors.New("Theme path exists but is not a directory")
	}

	return errors.Wrap(runGit(ctx, nil, "-C", themePath, "pull"), "Error while pulling in theme dir")
}

func (w *Worker) generateConfigFile(configFilePath string, blog userstore.Blog) error {
	configFile := struct {
		BuildFuture            bool     `json:"buildFuture"`
		DisableKinds           []string `json:"disableKinds"`
		EnableInlineShortcodes bool     `json:"enableInlineShortcodes"`
		LanguageCode           string   `json:"languageCode"`
		RSSLimit               uint     `json:"rssLimit"`
		Title                  string   `json:"title"`
	}{
		BuildFuture:            true,
		DisableKinds:           []string{"section", "taxonomy", "taxonomyTerm", "sitemap", "robotsTXT", "404"},
		EnableInlineShortcodes: false,
		LanguageCode:           "en-us",
		RSSLimit:               100,
		Title:                  blog.DisplayName,
	}

	configFileData, err := json.Marshal(&configFile)

	if err != nil {
		return errors.Wrap(err, "Error while encoding configuration file")
	}

	if err := ioutil.WriteFile(configFilePath, configFileData, 0600); err != nil {
		return errors.Wrap(err, "Error while writing config file")
	}

	return nil
}

func (w *Worker) runHugo(ctx context.Context, configFilePath string) error {
	destDirPath := path.Join(w.workDir, resultDirectory)

	if err := os.RemoveAll(destDirPath); err != nil {
		return errors.Wrap(err, "Error while cleaning destination directory")
	}

	hugoPath, err := exec.LookPath("hugo")

	if err != nil {
		return errors.Wrap(err, "Cannot find hugo in PATH")
	}

	var stderrBuffer bytes.Buffer
	args := []string{
		"--config", configFilePath,
		"--source", path.Join(w.workDir, blogDirectory),
		"--destination", destDirPath,
		"--themesDir", w.workDir,
		"--theme", themeDirectory,
	}
	hugoCmd := exec.CommandContext(ctx, hugoPath, args...)
	hugoCmd.Env = []string{}
	hugoCmd.Stdin = nil
	hugoCmd.Stdout = nil
	hugoCmd.Stderr = &stderrBuffer

	log.Printf("Running hugo %v", args)

	if err := hugoCmd.Run(); err != nil {
		return errors.Wrapf(err, "Hugo returned an error (stderr: %s)", strings.TrimSpace(stderrBuffer.String()))
	}

	return nil
}

func (w *Worker) uploadHTMLFiles(ctx context.Context, username, slug string) error {
	sourceDir := path.Join(w.workDir, resultDirectory)
	bucket, err := blob.OpenBucket(ctx, w.blogOutputURL)

	if err != nil {
		return errors.Wrap(err, "Error while opening bucket")
	}

	copyFunc := func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		key := path.Join(username, slug, p[len(sourceDir):])

		if info.IsDir() {
			return nil // there are no directories in blob stores
		} else if info.Mode().IsRegular() {
			err = uploadFile(ctx, bucket, key, p)
		} else {
			err = errors.Errorf("Don't know how to copy %s: unknown file type", p)
		}

		return err
	}

	return errors.Wrap(filepath.Walk(sourceDir, copyFunc), "Error while copying files")
}

func uploadFile(ctx context.Context, bucket *blob.Bucket, key, srcPath string) (err error) {
	srcFd, err := os.OpenFile(srcPath, os.O_RDONLY, 0)

	if err != nil {
		return errors.Wrap(err, "Error while opening source file")
	}

	defer srcFd.Close()

	writer, err := bucket.NewWriter(ctx, key, nil)

	if err != nil {
		return errors.Wrap(err, "Error while creating writer")
	}

	if _, err = io.Copy(writer, srcFd); err != nil {
		return errors.Wrap(err, "Error while copying file data")
	}

	if err = errors.Wrap(writer.Close(), "Error while finalizing upload"); err != nil {
		return errors.Wrap(err, "Error while finalizing upload")
	}

	return nil
}
