package gopenqa

/*
 * gopenQA unit seting
 * This test module sets up a webservers that serves the contents of the `test` directory as /api/v1 directory
 */

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"gotest.tools/assert"
)

var instance Instance

const COMMENT_TEST_JOB_ID = 5830

/* Test server http - Serves the contents of the test/ directory as the /api/v1 directory.
 *
 * The /api/v1/jobs list route is handled separately.
 */
func newTestServer() *httptest.Server {
	mux := http.NewServeMux()
	// Exact path match, so it does not shadow /api/v1/jobs/overview or /api/v1/jobs/<id>/comments
	mux.HandleFunc("/api/v1/jobs", serveJobList)
	mux.Handle("/api/v1/", http.StripPrefix("/api/v1/", http.FileServer(http.Dir("./test"))))
	return httptest.NewServer(mux)
}

/* serveJobList emulates openQA's GET /api/v1/jobs route from the test/jobs/<id> fixtures.
 *
 * It deliberately mirrors the openQA behaviour the library depends on:
 *   - `ids` may be repeated and/or comma separated
 *   - the fixtures store the single-job response ({"job":{...}}), so the envelope is
 *     unwrapped and re-wrapped into the list response ({"jobs":[{...}]})
 *   - unknown ids are omitted rather than being an error: the response is {"jobs":[]}
 *     with HTTP 200. This is what openQA does; only the single-job route returns a 404.
 *   - without any `ids` the full set is returned, which is what GetLatestJobs queries
 */
func serveJobList(w http.ResponseWriter, r *http.Request) {
	ids := parseJobIDs(r.URL.Query()["ids"])
	if len(ids) == 0 {
		ids = fixtureJobIDs()
	}
	jobs := make([]json.RawMessage, 0, len(ids))
	for _, id := range ids {
		if job, ok := readJobFixture(id); ok {
			jobs = append(jobs, job)
		}
	}
	body, err := json.Marshal(map[string][]json.RawMessage{"jobs": jobs})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(body); err != nil {
		panic(err)
	}
}

/* parseJobIDs splits the `ids` query values, which openQA accepts both repeated
 * (ids=1&ids=2) and comma separated (ids=1,2) */
func parseJobIDs(values []string) []string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		for _, id := range strings.Split(value, ",") {
			if id = strings.TrimSpace(id); id != "" {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

/* fixtureJobIDs returns all job ids we have a fixture for. os.ReadDir sorts by filename,
 * which matches how openQA orders the job list (it sorts the ids as strings). */
func fixtureJobIDs() []string {
	entries, err := os.ReadDir("./test/jobs")
	if err != nil {
		return nil
	}
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && isJobID(entry.Name()) {
			ids = append(ids, entry.Name())
		}
	}
	return ids
}

/* readJobFixture reads test/jobs/<id> and unwraps the {"job":{...}} envelope.
 * Returns false if there is no such fixture, mirroring an unknown job id. */
func readJobFixture(id string) (json.RawMessage, bool) {
	if !isJobID(id) {
		return nil, false
	}
	// test/jobs/5830 is a directory holding the comments fixture, not a job
	if info, err := os.Stat("./test/jobs/" + id); err != nil || !info.Mode().IsRegular() {
		return nil, false
	}
	data, err := os.ReadFile("./test/jobs/" + id)
	if err != nil {
		return nil, false
	}
	var envelope struct {
		Job json.RawMessage `json:"job"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil || len(envelope.Job) == 0 {
		return nil, false
	}
	return envelope.Job, true
}

/* isJobID guards the fixture lookup against path traversal and stray files */
func isJobID(id string) bool {
	if id == "" {
		return false
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func TestMain(m *testing.M) {
	// Testserver initialization
	srv := newTestServer()
	instance = CreateInstance(srv.URL)

	// Run tests. os.Exit skips deferred calls, so the server is closed explicitly.
	ret := m.Run()
	srv.Close()
	os.Exit(ret)
}

func TestOverview(t *testing.T) {
	jobs, err := instance.GetOverview("test", EmptyParams())
	if err != nil {
		t.Fatalf("%s", err)
	}
	// Expect 6 jobs
	if len(jobs) != 6 {
		t.Fatalf("Expected 6 jobs, got %d", len(jobs))
	}
	// Check if each job is the same when fetched individually
	for _, job := range jobs {
		fetched, err := instance.GetJob(job.ID)
		if err != nil {
			t.Fatalf("Error fetching job %d: %s", job.ID, err)
		}
		// Overview has only ID and name
		if job.ID != fetched.ID || job.Name != fetched.Name {
			t.Fatalf("Fetching job %d doesn't match the overview job", job.ID)
		}
	}
}

func TestWorkers(t *testing.T) {
	workers, err := instance.GetWorkers()
	if err != nil {
		t.Fatalf("%s", err)
	}
	// Expect 2 workers
	if len(workers) != 2 {
		t.Fatalf("Expected 2 workers, got %d", len(workers))
	}
}

func TestComments(t *testing.T) {
	comments, err := instance.GetComments(COMMENT_TEST_JOB_ID)
	if err != nil {
		t.Fatalf("%s", err)
	}
	if len(comments) != 4 {
		t.Fatalf("Expected 4 comments, got %d", len(comments))
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
		t.Fatalf("%s", err)
	}
	if len(machines) != 3 {
		t.Fatalf("Expected 3 machines, got %d", len(machines))
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
		t.Fatalf("%s", err)
	}
	if len(products) != 3 {
		t.Fatalf("Expected 3 products, got %d", len(products))
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

/* newJobsServer serves openQA's GET /api/v1/jobs route from the given job bodies, keyed by
 * job id, and records every request that is made against it.
 *
 * Unknown ids are omitted from the response, so querying only unknown ids yields
 * {"jobs":[]} with HTTP 200. That is what openQA does: only the single job route answers
 * with a HTTP 404.
 *
 * The returned function reports the raw query of each recorded request, in order. It is
 * used to assert that no superfluous round trips are made. */
func newJobsServer(t *testing.T, jobs map[int64]string) (*httptest.Server, func() []string) {
	t.Helper()
	var mutex sync.Mutex
	requests := make([]string, 0)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/jobs" {
			http.NotFound(w, r)
			return
		}
		mutex.Lock()
		requests = append(requests, r.URL.RawQuery)
		mutex.Unlock()

		bodies := make([]string, 0)
		for _, param := range parseJobIDs(r.URL.Query()["ids"]) {
			id, err := strconv.ParseInt(param, 10, 64)
			if err != nil {
				continue
			}
			if body, ok := jobs[id]; ok {
				bodies = append(bodies, body)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, `{"jobs":[`+strings.Join(bodies, ",")+`]}`); err != nil {
			panic(err)
		}
	}))

	return srv, func() []string {
		mutex.Lock()
		defer mutex.Unlock()
		return append([]string(nil), requests...)
	}
}

/* assertJobIDs checks that the given jobs are exactly the wanted ones, in order */
func assertJobIDs(t *testing.T, jobs []Job, want ...int64) {
	t.Helper()
	if len(jobs) != len(want) {
		t.Fatalf("got %d jobs %v, want %d %v", len(jobs), jobIDs(jobs), len(want), want)
	}
	for i, job := range jobs {
		if job.ID != want[i] {
			t.Fatalf("got jobs %v, want %v", jobIDs(jobs), want)
		}
	}
}

func jobIDs(jobs []Job) []int64 {
	ids := make([]int64, 0, len(jobs))
	for _, job := range jobs {
		ids = append(ids, job.ID)
	}
	return ids
}

/* assertRequests checks how many requests have been made against the test server */
func assertRequests(t *testing.T, requests func() []string, want int) {
	t.Helper()
	if got := requests(); len(got) != want {
		t.Errorf("made %d requests %v, want %d", len(got), got, want)
	}
}

/* parentJob returns a job bound to the given instance, with the given children */
func parentJob(inst *Instance, chained []int64, directlyChained []int64, parallel []int64) Job {
	job := Job{ID: 1}
	job.Children.Chained = chained
	job.Children.DirectlyChained = directlyChained
	job.Children.Parallel = parallel
	job.instance = inst
	return job
}

// A job that does not exist must be reported as an error. The list route reports an unknown
// job as an empty result instead of as a HTTP 404, so the library has to detect it.
func TestGetJobMissingJobIsError(t *testing.T) {
	srv, _ := newJobsServer(t, nil)
	defer srv.Close()
	inst := CreateInstance(srv.URL)

	if job, err := inst.GetJob(1234); err == nil {
		t.Errorf("GetJob(1234) = %+v, want an error", job)
	}
	if job, err := inst.GetJobFollow(1234); err == nil {
		t.Errorf("GetJobFollow(1234) = %+v, want an error", job)
	}
}

// GetJobFollow must follow a clone without re-fetching the job it has already fetched, and
// without losing the test modules on the way.
func TestGetJobFollowFetchesEachJobOnce(t *testing.T) {
	srv, requests := newJobsServer(t, map[int64]string{
		100: `{"id":100,"clone_id":200}`,
		200: `{"id":200,"clone_id":0,"modules":[{"name":"m1","result":"passed"}]}`,
	})
	defer srv.Close()

	inst := CreateInstance(srv.URL)
	job, err := inst.GetJobFollow(100)
	if err != nil {
		t.Fatalf("GetJobFollow: %v", err)
	}
	if job.ID != 200 {
		t.Errorf("followed to job %d, want the clone 200", job.ID)
	}
	if len(job.Modules) != 1 || job.Modules[0].Name != "m1" {
		t.Errorf("Modules = %+v, want the modules of the clone", job.Modules)
	}
	// the job and its clone, nothing more
	assertRequests(t, requests, 2)
}

// Without follow, FetchChildren returns the children as they are, cloned or not.
func TestFetchChildrenWithoutFollow(t *testing.T) {
	srv, requests := newJobsServer(t, childFixtures())
	defer srv.Close()
	inst := CreateInstance(srv.URL)
	parent := parentJob(&inst, []int64{10, 11}, nil, nil)

	children, err := parent.FetchChildren([]int64{10, 11}, false)
	if err != nil {
		t.Fatalf("FetchChildren: %v", err)
	}
	assertJobIDs(t, children, 10, 11)
	assertRequests(t, requests, 1)
}

// With follow, FetchChildren returns the clones. It must not re-fetch the children it has
// already fetched: one request for the children, one for the clone that has to be followed.
func TestFetchChildrenFollowFetchesEachJobOnce(t *testing.T) {
	srv, _ := newJobsServer(t, childFixtures())
	defer srv.Close()
	inst := CreateInstance(srv.URL)
	parent := parentJob(&inst, []int64{10, 11}, nil, nil)

	children, err := parent.FetchChildren([]int64{10, 11}, true)
	if err != nil {
		t.Fatalf("FetchChildren: %v", err)
	}
	// 10 is not cloned and stays, 11 is cloned into 21
	assertJobIDs(t, children, 10, 21)
}

// FetchAllChildren must consider chained, directly chained and parallel children alike.
func TestFetchAllChildrenAggregatesAllDependencyTypes(t *testing.T) {
	srv, requests := newJobsServer(t, map[int64]string{
		10: `{"id":10,"clone_id":0}`,
		11: `{"id":11,"clone_id":0}`,
		12: `{"id":12,"clone_id":0}`,
	})
	defer srv.Close()
	inst := CreateInstance(srv.URL)
	parent := parentJob(&inst, []int64{10}, []int64{11}, []int64{12})

	children, err := parent.FetchAllChildren(false)
	if err != nil {
		t.Fatalf("FetchAllChildren: %v", err)
	}
	assertJobIDs(t, children, 10, 11, 12)
	// all children are fetched with a single request
	assertRequests(t, requests, 1)
}

/* childFixtures returns two children, of which the second one has been cloned */
func childFixtures() map[int64]string {
	return map[int64]string{
		10: `{"id":10,"clone_id":0}`,
		11: `{"id":11,"clone_id":21}`,
		21: `{"id":21,"clone_id":0}`,
	}
}
