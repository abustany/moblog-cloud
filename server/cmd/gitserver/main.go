package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/abustany/moblog-cloud/pkg/gitserver"
	"github.com/abustany/moblog-cloud/pkg/workqueue"
)

func main() {
	listenAddress := flag.String("listen", "127.0.0.1:8080", "Address to listen on, of the form IP:PORT")
	baseAPIPath := flag.String("baseAPIPath", "", "Path under which to serve the API")
	repositoryBase := flag.String("repositoryBase", "", "Base path where user repositories are stored")
	adminServerURL := flag.String("adminServer", "", "URL to the admin server")
	redisJobQueueURL := flag.String("redisJobQueue", "", "URL to the Redis server to use for the job queue")

	flag.Parse()

	if *repositoryBase == "" {
		log.Fatalf("Missing option: -repositoryBase")
	}

	if *adminServerURL == "" {
		log.Fatalf("Missing option: -adminServer")
	}

	var jobQueue workqueue.Queue
	var err error

	if *redisJobQueueURL == "" {
		log.Printf("Warning: using an in-memory work queue, render jobs will not be triggered")
		jobQueue, err = workqueue.NewMemoryQueue()

		if err == nil {
			defer jobQueue.(*workqueue.MemoryQueue).Stop()
		}
	} else {
		jobQueue, err = workqueue.NewRedisQueue(*redisJobQueueURL)

		if err == nil {
			defer jobQueue.(*workqueue.RedisQueue).Stop()
		}
	}

	if err != nil {
		log.Fatalf("Error while creating job queue: %s", err)
	}

	s, err := gitserver.New(*baseAPIPath, *repositoryBase, *adminServerURL, jobQueue)

	if err != nil {
		log.Fatalf("Error while creating gitserver: %s", err)
	}

	log.Printf("Listening on %s", *listenAddress)
	err = http.ListenAndServe(*listenAddress, s)

	log.Fatalf("Error listening on %s: %s", *listenAddress, err)
}
