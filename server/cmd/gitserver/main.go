package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/abustany/moblog-cloud/pkg/gitserver"
)

var listenAddress = flag.String("listen", "127.0.0.1:8080", "Address to listen on, of the form IP:PORT")
var repositoryBase = flag.String("repositoryBase", "", "Base path where user repositories are stored")
var templateRepository = flag.String("templateRepository", "", "Path to the template repository for new repositories")
var adminServerURL = flag.String("adminServer", "", "URL to the admin server")

func main() {
	flag.Parse()

	if *repositoryBase == "" {
		log.Fatalf("Missing option: -repositoryBase")
	}

	if *templateRepository == "" {
		log.Fatalf("Missing option: -templateRepository")
	}

	if *adminServerURL == "" {
		log.Fatalf("Missing option: -adminServer")
	}

	s, err := gitserver.New(*repositoryBase, *templateRepository, *adminServerURL)

	if err != nil {
		log.Fatalf("Error while creating gitserver: %s", err)
	}

	log.Printf("Listening on %s", *listenAddress)
	err = http.ListenAndServe(*listenAddress, s)

	log.Fatalf("Error listening on %s: %s", *listenAddress, err)
}
