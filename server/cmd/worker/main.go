package main

import (
	"flag"
	"log"

	"github.com/abustany/moblog-cloud/pkg/worker"
	"github.com/abustany/moblog-cloud/pkg/workqueue"
)

var redisJobQueueURL = flag.String("redisJobQueue", "", "URL to the Redis server to use for the job queue")
var adminServerURL = flag.String("adminServer", "", "URL of the admin server")
var gitServerURL = flag.String("gitServer", "", "URL of the git server")
var workDir = flag.String("workDir", "", "Directory where to checkout the blog source and do the rendering work")
var themeRepositoryURL = flag.String("themeRepository", "", "URL of the Git repository holding the blog theme")

func main() {
	flag.Parse()

	if *redisJobQueueURL == "" {
		log.Fatalf("Missing option: -redisJobQueue")
	}

	if *adminServerURL == "" {
		log.Fatalf("Missing option: -adminServer")
	}

	if *gitServerURL == "" {
		log.Fatalf("Missing option: -gitServer")
	}

	if *workDir == "" {
		log.Fatalf("Missing option: -workDir")
	}

	if *themeRepositoryURL == "" {
		log.Fatalf("Missing option: -themeRepository")
	}

	queue, err := workqueue.NewRedisQueue(*redisJobQueueURL)

	if err != nil {
		log.Fatalf("Error while initializing Redis work queue: %s", err)
	}

	defer queue.Stop()

	worker, err := worker.New(queue, *adminServerURL, *gitServerURL, *workDir, *themeRepositoryURL)

	if err != nil {
		log.Fatalf("Error while initializing worker: %s", err)
	}

	defer worker.Stop()
}
