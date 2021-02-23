package gopenqa

/*
 * gopenQA unit seting
 * This test module sets up a webservers that serves the contents of the `test` directory as /api/v1 directory
 */

import (
	"log"
	"net/http"
	"os"
	"testing"
)

var instance Instance

func setupTestServer() {
	fs := http.FileServer(http.Dir("./test"))
	http.Handle("/api/v1/", http.StripPrefix("/api/v1/", fs))
	go func() {
		if err := http.ListenAndServe(":8421", nil); err != nil {
			panic(err)
		}
	}()
}

func TestMain(m *testing.M) {
	// Testserver initialization
	setupTestServer()
	log.Println("http server setup complete")
	instance = CreateInstance("http://localhost:8421")

	// Run tests
	ret := m.Run()
	os.Exit(ret)
}

func TestOverview(t *testing.T) {
	jobs, err := instance.GetOverview("test", EmptyParams())
	if err != nil {
		log.Fatalf("%s", err)
		return
	}
	// Expect 5 jobs
	if len(jobs) != 6 {
		log.Fatalf("Expected 6 jobs, got %d", len(jobs))
		return
	}
}

func TestWorkers(t *testing.T) {
	workers, err := instance.GetWorkers()
	if err != nil {
		log.Fatalf("%s", err)
		return
	}
	// Expect 2 workers
	if len(workers) != 2 {
		log.Fatalf("Expected 2 workers, got %d", len(workers))
		return
	}
}
