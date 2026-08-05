package gopenqa

/*
 * gopenQA unit seting
 * This test module sets up a webservers that serves the contents of the `test` directory as /api/v1 directory
 */

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

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

func TestMachines(t *testing.T) {
	machines, err := instance.GetMachines()
	if err != nil {
		log.Fatalf("%s", err)
		return
	}
	if len(machines) != 3 {
		log.Fatalf("Expected 3 machines, got %d", len(machines))
		return
	}
	assert.Equal(t, machines[0].ID, 1)
	assert.Equal(t, machines[0].Backend, "qemu")
	assert.Equal(t, machines[0].Name, "worker1")
	assert.Equal(t, machines[0].Settings["HDDSIZEGB"], "20")
	assert.Equal(t, machines[1].ID, 2)
	assert.Equal(t, machines[1].Backend, "qemu")
	assert.Equal(t, machines[1].Name, "worker2")
	assert.Equal(t, machines[1].Settings["HDDSIZEGB"], "30")
	assert.Equal(t, machines[2].ID, 4)
	assert.Equal(t, machines[2].Backend, "qemu")
	assert.Equal(t, machines[2].Name, "worker4")
	assert.Equal(t, machines[2].Settings["HDDSIZEGB"], "10")
}

func TestProduct(t *testing.T) {
	products, err := instance.GetProducts()
	if err != nil {
		log.Fatalf("%s", err)
		return
	}
	if len(products) != 3 {
		log.Fatalf("Expected 3 products, got %d", len(products))
		return
	}
	assert.Equal(t, products[0].ID, 1)
	assert.Equal(t, products[0].Arch, "x86_64")
	assert.Equal(t, products[0].Distri, "opensuse")
	assert.Equal(t, products[0].Flavor, "DVD")
	assert.Equal(t, products[0].Settings["QEMURAM"], "2048")
	assert.Equal(t, products[0].Settings["HDD_1"], "openSUSE-1-DVD.iso")
	assert.Equal(t, products[1].ID, 2)
	assert.Equal(t, products[1].Arch, "x86_64")
	assert.Equal(t, products[1].Distri, "opensuse")
	assert.Equal(t, products[1].Flavor, "Image")
	assert.Equal(t, products[1].Settings["STAGING"], "1")
	assert.Equal(t, products[2].ID, 3)
	assert.Equal(t, products[2].Arch, "aarch64")
	assert.Equal(t, products[2].Distri, "opensuse")
	assert.Equal(t, products[2].Flavor, "DVD")
	assert.Equal(t, products[2].Settings["BOOT_HDD_IMAGE"], "1")
	assert.Equal(t, products[2].Settings["HDD_1"], "openSUSE-1-aarch64-DVD.iso")
}

// GHSA-rwxw-gmm3-whv5: CreateInstance must install safe non-zero defaults.
func TestCreateInstanceDefaultLimits(t *testing.T) {
	inst := CreateInstance("http://example.invalid")
	if inst.httpTimeout != DefaultHTTPTimeout {
		t.Fatalf("httpTimeout: got %v, want %v", inst.httpTimeout, DefaultHTTPTimeout)
	}
	if inst.maxResponseBytes != DefaultMaxResponseBytes {
		t.Fatalf("maxResponseBytes: got %d, want %d", inst.maxResponseBytes, DefaultMaxResponseBytes)
	}
}

// GHSA-rwxw-gmm3-whv5: response bodies larger than the configured limit must fail
// without an unbounded io.ReadAll of the full body.
func TestRequestRejectsOversizedResponseBody(t *testing.T) {
	const maxBytes int64 = 1024

	t.Run("maxBytes+1", func(t *testing.T) {
		body := strings.Repeat("x", int(maxBytes)+1)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, body)
		}))
		defer srv.Close()

		inst := CreateInstance(srv.URL)
		inst.SetMaxResponseBytes(maxBytes)

		buf, err := inst.get(srv.URL+"/api/v1/jobs/1", nil)
		if err == nil {
			t.Fatal("expected error for body of maxBytes+1, got nil")
		}
		if !strings.Contains(err.Error(), "maximum size") {
			t.Fatalf("expected maximum size error, got: %v", err)
		}
		if len(buf) != 0 {
			t.Fatalf("expected empty buffer on oversize, got %d bytes", len(buf))
		}
	})

	// Streaming multi-MiB body: a broken "ReadAll then check len" implementation
	// would allocate the full payload; LimitReader fails after maxBytes+1 quickly.
	t.Run("streaming_large_body", func(t *testing.T) {
		const offerBytes = 8 << 20 // 8 MiB offered; limit is 1 KiB
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			chunk := strings.Repeat("x", 32<<10)
			written := 0
			for written < offerBytes {
				n, err := io.WriteString(w, chunk)
				written += n
				if err != nil {
					return
				}
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}
		}))
		defer srv.Close()

		inst := CreateInstance(srv.URL)
		inst.SetMaxResponseBytes(maxBytes)
		inst.SetHTTPTimeout(5 * time.Second)

		start := time.Now()
		buf, err := inst.get(srv.URL+"/api/v1/jobs/1", nil)
		elapsed := time.Since(start)

		if err == nil {
			t.Fatal("expected error for streaming oversized body, got nil")
		}
		if !strings.Contains(err.Error(), "maximum size") {
			t.Fatalf("expected maximum size error, got: %v", err)
		}
		if len(buf) != 0 {
			t.Fatalf("expected empty buffer on oversize, got %d bytes", len(buf))
		}
		// Must fail from the size cap, not only after reading multi-MiB then timing out.
		if elapsed >= 2*time.Second {
			t.Fatalf("oversize rejection took %v; likely unbounded body read", elapsed)
		}
	})
}

// GHSA-rwxw-gmm3-whv5: a stalling server must not block the client indefinitely.
func TestRequestTimesOutOnStallingServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Stall until the client cancels (Timeout) or a long safety deadline.
		select {
		case <-r.Context().Done():
			return
		case <-time.After(10 * time.Second):
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"job":{}}`)
	}))
	defer srv.Close()

	inst := CreateInstance(srv.URL)
	inst.SetHTTPTimeout(200 * time.Millisecond)

	start := time.Now()
	_, err := inst.get(srv.URL+"/api/v1/jobs/1", nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("expected net.Error timeout, got: %v", err)
	}
	// Must fail well before the handler's safety deadline.
	if elapsed >= 1500*time.Millisecond {
		t.Fatalf("request took %v, expected client-side timeout well under 1.5s", elapsed)
	}
}

// Bodies exactly at the limit must still succeed (limit is inclusive).
func TestRequestAcceptsBodyAtExactLimit(t *testing.T) {
	const maxBytes int64 = 128
	body := strings.Repeat("a", int(maxBytes))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, body)
	}))
	defer srv.Close()

	inst := CreateInstance(srv.URL)
	inst.SetMaxResponseBytes(maxBytes)

	buf, err := inst.get(srv.URL+"/api/v1/jobs/1", nil)
	if err != nil {
		t.Fatalf("unexpected error for body at exact limit: %v", err)
	}
	if string(buf) != body {
		t.Fatalf("body mismatch: got %d bytes, want %d", len(buf), len(body))
	}
}

// Non-positive config must fall back to safe defaults at request time
// (not "unlimited" and not "reject everything").
func TestRequestNonPositiveConfigFallsBackToDefaults(t *testing.T) {
	body := `{"ok":true}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, body)
	}))
	defer srv.Close()

	inst := CreateInstance(srv.URL)
	inst.SetMaxResponseBytes(0)
	inst.SetHTTPTimeout(0)

	buf, err := inst.get(srv.URL+"/api/v1/jobs/1", nil)
	if err != nil {
		t.Fatalf("expected success with non-positive config (defaults apply): %v", err)
	}
	if string(buf) != body {
		t.Fatalf("body mismatch: got %q, want %q", buf, body)
	}
}

// GetJobsFollow must preserve Job.Modules when following a cloned job. The clone hop
// used to go through the single-job endpoint (which never returns Modules, see GetJob's
// doc comment), silently losing them.
func TestGetJobsFollowPreservesModules(t *testing.T) {
	const originalID, cloneID = 100, 200
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/jobs" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		switch r.URL.Query().Get("ids") {
		case "100":
			_, _ = io.WriteString(w, `{"jobs":[{"id":100,"clone_id":200}]}`)
		case "200":
			_, _ = io.WriteString(w, `{"jobs":[{"id":200,"clone_id":0,"modules":[{"name":"m1","category":"c","result":"passed","flags":["important"]}]}]}`)
		default:
			_, _ = io.WriteString(w, `{"jobs":[]}`)
		}
	}))
	defer srv.Close()

	inst := CreateInstance(srv.URL)
	jobs, err := inst.GetJobsFollow([]int64{originalID})
	if err != nil {
		t.Fatalf("GetJobsFollow failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	job := jobs[0]
	if job.ID != cloneID {
		t.Fatalf("expected followed job id %d, got %d", cloneID, job.ID)
	}
	if len(job.Modules) != 1 || job.Modules[0].Name != "m1" {
		t.Fatalf("expected Modules to be preserved after following clone, got %+v", job.Modules)
	}
}
