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

	"gotest.tools/assert"
)

var instance Instance

const COMMENT_TEST_JOB_ID = 5830

/* Test server http - Serves directories in test/ */
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
	// Expect 6 jobs
	if len(jobs) != 6 {
		log.Fatalf("Expected 6 jobs, got %d", len(jobs))
		return
	}
	// Check if each job is the same when fetched individually
	for _, job := range jobs {
		fetched, err := instance.GetJob(job.ID)
		if err != nil {
			log.Fatalf("Error fetching job %d: %s", job.ID, err)
			return
		}
		// Overview has only ID and name
		if job.ID != fetched.ID || job.Name != fetched.Name {
			log.Fatalf("Fetching job %d doesn't match the overview job", job.ID)
			return
		}
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

func TestComments(t *testing.T) {
	comments, err := instance.GetComments(COMMENT_TEST_JOB_ID)
	if err != nil {
		log.Fatalf("%s", err)
		return
	}
	if len(comments) != 4 {
		log.Fatalf("Expected 4 comments, got %d", len(comments))
		return
	}
	// Check comments for expected content
	for _, comment := range comments {
		assert.Equal(t, comment.User, "phoenix")
	}
	assert.Equal(t, comments[0].ID, 14)
	assert.Equal(t, comments[1].ID, 15)
	assert.Equal(t, comments[2].ID, 16)
	assert.Equal(t, comments[3].ID, 17)
	assert.Equal(t, comments[1].Text, "Comment 2.")
	assert.Equal(t, comments[2].Text, "poo#42")
	assert.Equal(t, comments[3].Text, "bsc#1337")
	assert.Assert(t, len(comments[2].BugRefs) == 1)
	assert.Assert(t, len(comments[3].BugRefs) == 1)
	assert.Assert(t, comments[2].BugRefs[0] == "poo#42")
	assert.Assert(t, comments[3].BugRefs[0] == "bsc#1337")

}
