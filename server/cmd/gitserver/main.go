package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/abustany/moblog-cloud/pkg/gitserver"
	"github.com/abustany/moblog-cloud/pkg/workqueue"
)

var listenAddress = flag.String("listen", "127.0.0.1:8080", "Address to listen on, of the form IP:PORT")
var repositoryBase = flag.String("repositoryBase", "", "Base path where user repositories are stored")
var adminServerURL = flag.String("adminServer", "", "URL to the admin server")
var redisJobQueueURL = flag.String("redisJobQueue", "", "URL to the Redis server to use for the job queue")

func main() {
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

	s, err := gitserver.New(*repositoryBase, *adminServerURL, jobQueue)

	if err != nil {
		log.Fatalf("Error while creating gitserver: %s", err)
	}

	log.Printf("Listening on %s", *listenAddress)
	err = http.ListenAndServe(*listenAddress, s)

	log.Fatalf("Error listening on %s: %s", *listenAddress, err)
}
