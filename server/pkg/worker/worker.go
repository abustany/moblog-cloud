package worker

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/abustany/moblog-cloud/pkg/jobs"
	"github.com/abustany/moblog-cloud/pkg/workqueue"
)

var PickTimeout = time.Second

type Worker struct {
	queue              workqueue.Queue
	adminServerURL     string
	gitServerURL       string
	workDir            string
	themeRepositoryURL string
	blogOutputURL      string
	closeChannel       chan chan struct{}
}

func New(queue workqueue.Queue, adminServerURL, gitServerURL, workDir, themeRepositoryURL, blogOutputURL string) (*Worker, error) {
	w := &Worker{
		queue:              queue,
		adminServerURL:     adminServerURL,
		gitServerURL:       gitServerURL,
		workDir:            workDir,
		themeRepositoryURL: themeRepositoryURL,
		blogOutputURL:      blogOutputURL,
		closeChannel:       make(chan chan struct{}),
	}

	go w.consumeJobs()

	return w, nil
}

func (w *Worker) Wait() {
	<-w.closeChannel
}

func (w *Worker) Stop() {
	ack := make(chan struct{})
	w.closeChannel <- ack
	<-ack
	close(w.closeChannel)
}

func (w *Worker) consumeJobs() {
	for {
		select {
		case ack := <-w.closeChannel:
			close(ack)
		default:
		}

		if err := w.consumeOneJob(); err != nil {
			log.Printf("Error while consuming job :%s", err)
		}
	}
}

func (w *Worker) consumeOneJob() error {
	job, err := w.queue.Pick(PickTimeout)

	if err != nil {
		return errors.Wrap(err, "Error while picking job")
	}

	if job == nil {
		return nil
	}

	log.Printf("Handling job %s", job.ID)

	ctx, cancel := context.WithTimeout(context.Background(), job.TTR)
	defer cancel()

	defer func() {
		if err := w.queue.Finish(job); err != nil {
			log.Printf("Error while finishing job %s: %s", job.ID, err)
		}
	}()

	if err := clearDirectory(w.workDir); err != nil {
		return errors.Wrap(err, "Error while clearing work directory")
	}

	switch jobData := job.Data.(type) {
	case jobs.RenderJob:
		log.Printf("Handling render job %+v", jobData)
		err = w.renderBlog(ctx, &jobData)
	default:
		return errors.Errorf("Unknown job type: %+v", job.Data)
	}

	if err != nil {
		log.Printf("Job %s failed: %s", job.ID, err)
	}

	return nil
}

func runGit(ctx context.Context, stdout io.Writer, args ...string) error {
	gitPath, err := exec.LookPath("git")

	if err != nil {
		return errors.Wrap(err, "Cannot find git in PATH")
	}

	var stderrBuffer bytes.Buffer
	gitCmd := exec.CommandContext(ctx, gitPath, args...)
	gitCmd.Env = []string{"GIT_TERMINAL_PROMPT=0"}
	gitCmd.Stdin = nil
	gitCmd.Stdout = stdout
	gitCmd.Stderr = &stderrBuffer

	log.Printf("Running git %v", args)

	if err := gitCmd.Run(); err != nil {
		return errors.Wrapf(err, "Git returned an error (stderr: %s)", strings.TrimSpace(stderrBuffer.String()))
	}

	return nil
}

func clearDirectory(dir string) error {
	entries, err := ioutil.ReadDir(dir)

	if err != nil {
		return errors.Wrap(err, "Error while listing directory")
	}

	for _, entry := range entries {
		if entry.Name() == "." || entry.Name() == ".." {
			continue
		}

		p := path.Join(dir, entry.Name())

		if err := os.RemoveAll(p); err != nil {
			return errors.Wrapf(err, "Error while deleting %s", p)
		}
	}

	return nil
}
