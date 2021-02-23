package gopenqa

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

/* Job instance */
type Job struct {
	AssignedWorkerID int      `json:"assigned_worker_id"`
	BlockedByID      int      `json:"blocked_by_id"`
	Children         Children `json:"children"`
	Parents          Children `json:"parents"`
	CloneID          int      `json:"clone_id"`
	GroupID          int      `json:"group_id"`
	ID               int      `json:"id"`
	// Modules
	Name string `json:"name"`
	// Parents
	Priority  int      `json:"priority"`
	Result    string   `json:"result"`
	Settings  Settings `json:"settings"`
	State     string   `json:"state"`
	Tfinished string   `json:"t_finished"`
	Tstarted  string   `json:"t_started"`
	Test      string   `json:"test"`
	/* this is added by the program and not part of the fetched json */
	Link   string
	Remote string
	Prefix string
}

type JobGroup struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	ParentID int    `json:"parent_id"`
}

/* Children struct is for chained, directly chained and parallel children/parents */
type Children struct {
	Chained         []int `json:"Chained"`
	DirectlyChained []int `json:"Directly chained"`
	Parallel        []int `json:"Parallel"`
}

/* Job Setting struct */
type Settings struct {
	Arch    string `json:"ARCH"`
	Backend string `json:"BACKEND"`
	Machine string `json:"MACHINE"`
}

/* Worker instance */
type Worker struct {
	Alive      int               `json:"alive"`
	Connected  int               `json:"connected"`
	Error      string            `json:"error"` // Error string if present
	Host       string            `json:"host"`
	ID         int               `json:"id"`
	Instance   int               `json:"instance"`
	Status     string            `json:"status"`
	Websocket  int               `json:"websocket"`
	Properties map[string]string `json:"properties"` // Worker properties as returned by openQA
}

/* Instance defines a openQA instance */
type Instance struct {
	URL string
}

/* Format job as a string */
func (j *Job) String() string {
	return fmt.Sprintf("%d %s (%s)", j.ID, j.Name, j.Test)
}
func (j *Job) JobState() string {
	if j.State == "done" {
		return j.Result
	}
	return j.State
}

func EmptyParams() map[string]string {
	return make(map[string]string, 0)
}

/* Create a openQA instance module */
func CreateInstance(url string) Instance {
	inst := Instance{URL: url}
	return inst
}

/* Create a openQA instance module for openqa.opensuse.org */
func CreateO3Instance() Instance {
	return CreateInstance("https://openqa.opensuse.org")
}

/* Query the job overview. params is a map for optional parameters, which will be added to the query.
 * Suitable parameters are `arch`, `distri`, `flavor`, `machine` or `arch`, but everything in this dict will be added to the url
 * Overview returns only the job id and name
 */
func (i *Instance) GetOverview(testsuite string, params map[string]string) ([]Job, error) {
	// Example values:
	// arch=x86_64
	// distri=sle
	// flavor=Server-DVD-Updates
	// machine=64bit

	// Build URL with all parameters
	url := fmt.Sprintf("%s/api/v1/jobs/overview", i.URL)
	if testsuite != "" {
		params["test"] = testsuite
	}
	if len(params) > 0 {
		url += "?" + mergeParams(params)
	}

	return fetchJobs(url)
}

/* Get only the latest jobs of a certain testsuite. Testsuite must be given here.
 * Additional parameters can be supplied via the params map (See GetOverview for more info about usage of those parameters)
 */
func (i *Instance) GetLatestJobs(testsuite string, params map[string]string) ([]Job, error) {
	// Expected result structure
	type ResultJob struct {
		Jobs []Job `json:"jobs"`
	}
	var jobs ResultJob
	if testsuite != "" {
		params["test"] = testsuite
	}
	url := fmt.Sprintf("%s/api/v1/jobs", i.URL)
	if testsuite != "" {
		params["test"] = testsuite
	}
	url += "?" + mergeParams(params)
	// Fetch jobs here, as we expect it to be in `jobs`
	r, err := http.Get(url)
	if err != nil {
		return jobs.Jobs, err
	}
	if r.StatusCode != 200 {
		return jobs.Jobs, fmt.Errorf("http status code %d", r.StatusCode)
	}
	err = json.NewDecoder(r.Body).Decode(&jobs)

	// Now, get only the latest job per group_id
	mapped := make(map[int]Job)
	for _, job := range jobs.Jobs {
		// TODO: Filter job results, if given

		// Only keep newer jobs (by ID) per group
		if f, ok := mapped[job.GroupID]; ok {
			if job.ID > f.ID {
				mapped[job.GroupID] = job
			}
		} else {
			mapped[job.GroupID] = job
		}
	}
	// Make slice from map
	ret := make([]Job, 0)
	for _, v := range mapped {
		ret = append(ret, v)
	}
	return ret, nil

}

// GetJob fetches detailled job information
func (i *Instance) GetJob(id int) (Job, error) {
	url := fmt.Sprintf("%s/api/v1/jobs/%d", i.URL, id)
	job, err := fetchJob(url)
	job.Link = fmt.Sprintf("%s/tests/%d", i.URL, id)
	return job, err
}

func (i *Instance) GetJobGroups() ([]JobGroup, error) {
	url := fmt.Sprintf("%s/api/v1/job_groups", i.URL)
	return fetchJobGroups(url)
}

func (i *Instance) GetWorkers() ([]Worker, error) {
	url := fmt.Sprintf("%s/api/v1/workers", i.URL)
	return fetchWorkers(url)
}

func fetchJobs(url string) ([]Job, error) {
	jobs := make([]Job, 0)
	r, err := http.Get(url)
	if err != nil {
		return jobs, err
	}
	if r.StatusCode != 200 {
		return jobs, fmt.Errorf("http status code %d", r.StatusCode)
	}
	err = json.NewDecoder(r.Body).Decode(&jobs)
	return jobs, err
}

func fetchJobGroups(url string) ([]JobGroup, error) {
	jobs := make([]JobGroup, 0)
	r, err := http.Get(url)
	if err != nil {
		return jobs, err
	}
	if r.StatusCode != 200 {
		return jobs, fmt.Errorf("http status code %d", r.StatusCode)
	}
	err = json.NewDecoder(r.Body).Decode(&jobs)
	return jobs, err
}

func fetchWorkers(url string) ([]Worker, error) {
	r, err := http.Get(url)
	if err != nil {
		return make([]Worker, 0), err
	}
	if r.StatusCode != 200 {
		return make([]Worker, 0), fmt.Errorf("http status code %d", r.StatusCode)
	}
	// workers come in a "workers:[...]" dict
	workers := make(map[string][]Worker, 0)
	err = json.NewDecoder(r.Body).Decode(&workers)
	if workers, ok := workers["workers"]; ok {
		return workers, err
	}
	return make([]Worker, 0), nil
}

func fetchJob(url string) (Job, error) {
	// Expected result structure
	type ResultJob struct {
		Job Job `json:"job"`
	}
	var job ResultJob
	r, err := http.Get(url)
	if err != nil {
		return job.Job, err
	}
	if r.StatusCode != 200 {
		return job.Job, fmt.Errorf("http status code %d", r.StatusCode)
	}
	err = json.NewDecoder(r.Body).Decode(&job)
	return job.Job, err
}

/* merge given parameter string to URL parameters */
func mergeParams(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	vals := make([]string, 0)
	for k, v := range params {
		vals = append(vals, fmt.Sprintf("&%s=%s", k, v))
	}
	return strings.Join(vals, "&")[1:] // remove first '&'
}
