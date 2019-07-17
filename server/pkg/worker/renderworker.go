package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/abustany/moblog-cloud/pkg/adminserver"
	"github.com/abustany/moblog-cloud/pkg/jobs"
	"github.com/abustany/moblog-cloud/pkg/userstore"
)

func (w *Worker) renderBlog(ctx context.Context, job *jobs.RenderJob) error {
	adminClient, err := adminserver.NewClient(w.adminServerURL)

	if err != nil {
		return errors.Wrap(err, "Error while creating adminserver client")
	}

	blog, err := adminClient.GetUserBlog(job.Username, job.Repository)

	if err != nil {
		return errors.Wrap(err, "Error while fetching blog information")
	}

	authCookieFile := path.Join(w.workDir, "auth_cookie.txt")

	if err := ioutil.WriteFile(authCookieFile, []byte("Set-Cookie: "+job.AuthCookie+"\n"), 0600); err != nil {
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