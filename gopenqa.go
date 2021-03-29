package gopenqa

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

/* Instance defines a openQA instance */
type Instance struct {
	URL           string
	apikey        string
	apisecret     string
	verbose       bool
	maxRecursions int // Maximum number of recursions
}

// the settings are given as a bit of a weird dict:
// e.g. "settings":[{"key":"WORKER_CLASS","value":"\"plebs\""}]}]
// We create an internal struct to account for that
type weirdMachine struct {
	ID       int                 `json:"id"`
	Backend  string              `json:"backend"`
	Name     string              `json:"name"`
	Settings []map[string]string `json:"settings"`
}

func (mach *weirdMachine) CopySettingsFrom(src Machine) {
	mach.Settings = make([]map[string]string, 0)
	for k, v := range src.Settings {
		setting := make(map[string]string, 0)
		setting["key"] = k
		setting["value"] = v
		mach.Settings = append(mach.Settings, setting)
	}
}
func (mach *weirdMachine) CopySettingsTo(dst *Machine) {
	dst.Settings = make(map[string]string)
	for _, s := range mach.Settings {
		k, ok := s["key"]
		if !ok {
			continue
		}
		v, ok := s["value"]
		if !ok {
			continue
		}
		dst.Settings[k] = v
	}
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
	inst := Instance{URL: url, maxRecursions: 10, verbose: false}
	return inst
}

/* Create a openQA instance module for openqa.opensuse.org */
func CreateO3Instance() Instance {
	return CreateInstance("https://openqa.opensuse.org")
}

// Set the maximum allowed number of recursions before failing
func (i *Instance) SetMaxRecursionDepth(depth int) {
	i.maxRecursions = depth
}

// Set the API key and secret
func (i *Instance) SetApiKey(key string, secret string) {
	i.apikey = key
	i.apisecret = secret
}

func (i *Instance) SetVerbose(verbose bool) {
	i.verbose = verbose
}

func assignInstance(jobs []Job, instance *Instance) []Job {
	for i, j := range jobs {
		j.instance = instance
		jobs[i] = j
	}
	return jobs
}

func hmac_sha1(secret string, key string) []byte {
	h := hmac.New(sha1.New, []byte(key))
	h.Write([]byte(secret))
	return h.Sum(nil)
}

func url_path(url string) string {
	// Ignore http://
	url = strings.Replace(url, "http://", "", 1)
	url = strings.Replace(url, "https://", "", 1)
	// Path from first /
	i := strings.Index(url, "/")
	if i > 0 {
		return url[i:]
	}
	return url
}

/* Perform a POST request on the given url, and send the data as JSON if given
 * Add the APIKEY and APISECRET credentials, if given
 */
func (i *Instance) post(url string, data interface{}) ([]byte, error) {
	return i.request("POST", url, data)
}

/* Perform a request on the given url, and send the data as JSON if given
 * Add the APIKEY and APISECRET credentials, if given
 */
func (i *Instance) request(method string, url string, data interface{}) ([]byte, error) {
	buf := make([]byte, 0)
	if data != nil {
		var err error
		if buf, err = json.Marshal(data); err != nil {
			return buf, err
		}
		fmt.Printf("%s\n", string(buf)) // TODO: Remove this
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(buf))
	if err != nil {
		return make([]byte, 0), err
	}
	// Credentials are sent in the headers
	// "X-API-Key" -> api key
	// "X-API-Hash" -> sha1 hashed api secret
	// POST request
	if i.apikey != "" && i.apisecret != "" {
		req.Header.Add("X-API-Key", i.apikey)
		// The Hash gets salted with the timestamp
		// See https://github.com/os-autoinst/openQA-python-client/blob/master/src/openqa_client/client.py#L115
		// hmac_sha1_sum(/api/v1/machines1617024969, XXXXXXXXXXXXXXXXXX){
		timestamp := time.Now().Unix()
		req.Header.Add("X-API-Microtime", fmt.Sprintf("%d", timestamp))
		path := url_path(url)
		key := fmt.Sprintf("%s%d", path, timestamp)
		hash := fmt.Sprintf("%x", hmac_sha1(key, i.apisecret))
		req.Header.Add("X-API-Hash", hash)

	}
	// Perform request on a new http client
	c := http.Client{}
	r, err := c.Do(req)
	if err != nil {
		return make([]byte, 0), err
	}

	// First read body
	defer r.Body.Close()
	buf, err = ioutil.ReadAll(r.Body) // TODO: Limit read size
	if err != nil {
		return buf, err
	}

	// Check status code
	if r.StatusCode != 200 {
		if i.verbose {
			fmt.Fprintf(os.Stderr, "%s\n", string(buf))
		}
		return buf, fmt.Errorf("http status code %d", r.StatusCode)
	}
	return buf, nil
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

	jobs, err := fetchJobs(url)
	assignInstance(jobs, i)
	return jobs, err
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
		job.instance = i
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
	job.instance = i
	return job, err
}

// GetJob fetches detailled job information and follows the job, if it contains a CloneID
func (i *Instance) GetJobFollow(id int) (Job, error) {
	recursions := 0 // keep track of the number of recursions
fetch:
	url := fmt.Sprintf("%s/api/v1/jobs/%d", i.URL, id)
	job, err := fetchJob(url)
	if job.CloneID != 0 && job.CloneID != job.ID {
		recursions++
		if i.maxRecursions != 0 && recursions >= i.maxRecursions {
			return job, fmt.Errorf("maximum recusion depth reached")
		}
		id = job.CloneID
		goto fetch
	}
	job.Link = fmt.Sprintf("%s/tests/%d", i.URL, id)
	job.instance = i
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

func fetchJobTemplates(url string) ([]JobTemplate, error) {
	r, err := http.Get(url)
	if err != nil {
		return make([]JobTemplate, 0), err
	}
	if r.StatusCode != 200 {
		return make([]JobTemplate, 0), fmt.Errorf("http status code %d", r.StatusCode)
	}
	// the templates come as a "JobTemplates:[...]" dict
	templates := make(map[string][]JobTemplate, 0)
	err = json.NewDecoder(r.Body).Decode(&templates)
	if templates, ok := templates["JobTemplates"]; ok {
		return templates, err
	}
	return make([]JobTemplate, 0), nil
}

func fetchMachines(url string) ([]Machine, error) {
	r, err := http.Get(url)
	if err != nil {
		return make([]Machine, 0), err
	}
	if r.StatusCode != 200 {
		return make([]Machine, 0), fmt.Errorf("http status code %d", r.StatusCode)
	}

	// machines come as a "Machines:[...]" dict
	machines := make(map[string][]weirdMachine, 0)
	err = json.NewDecoder(r.Body).Decode(&machines)
	if machines, ok := machines["Machines"]; ok {
		// Parse those weird machines to actual machine instances
		ret := make([]Machine, 0)
		for _, mach := range machines {
			current := Machine{Name: mach.Name, Backend: mach.Backend, ID: mach.ID}
			mach.CopySettingsTo(&current)
			//buf, _ := json.Marshal(mach)
			//fmt.Println(string(buf))  // TODO: Remove me
			ret = append(ret, current)
		}
		return ret, err
	}
	return make([]Machine, 0), nil
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
		vals = append(vals, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(vals, "&")
}

/*
 * Fetch the given child jobs. Use with j.Children.Chained, j.Children.DirectlyChained and j.Children.Parallel
 * if follow is set to true, the method will return the cloned job instead of the original one, if present
 */
func (j *Job) FetchChildren(children []int, follow bool) ([]Job, error) {
	jobs := make([]Job, 0)
	for _, id := range children {
		job, err := j.instance.GetJobFollow(id)
		if err != nil {
			return jobs, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

/* Fetch all child jobs
 * follow determines if we should follow the given children, i.e. get their cloned jobs instead of the original ones if present
 */
func (j *Job) FetchAllChildren(follow bool) ([]Job, error) {
	children := make([]int, 0)
	children = append(children, j.Children.Chained...)
	children = append(children, j.Children.DirectlyChained...)
	children = append(children, j.Children.Parallel...)
	return j.FetchChildren(children, follow)
}

func (i *Instance) GetJobTemplates() ([]JobTemplate, error) {
	url := fmt.Sprintf("%s/api/v1/job_templates", i.URL)
	return fetchJobTemplates(url)
}

func (i *Instance) GetMachines() ([]Machine, error) {
	url := fmt.Sprintf("%s/api/v1/machines", i.URL)
	return fetchMachines(url)
}

func (i *Instance) GetMachine(id int) (Machine, error) {
	url := fmt.Sprintf("%s/api/v1/machines/%d", i.URL, id)
	if machines, err := fetchMachines(url); err != nil {
		return Machine{}, err
	} else {
		if len(machines) > 0 {
			return machines[0], nil
		} else {
			return Machine{}, nil
		}
	}
}

func (i *Instance) PostMachine(machine Machine) (Machine, error) {
	if i.apikey == "" || i.apisecret == "" {
		return Machine{}, fmt.Errorf("API key or secret not set")
	}

	var rurl string
	if machine.ID == 0 {
		rurl = fmt.Sprintf("%s/api/v1/machines", i.URL)
	} else {
		rurl = fmt.Sprintf("%s/api/v1/machines/%d", i.URL, machine.ID)
	}

	// Add parameters
	params := url.Values{}
	params.Add("backend", machine.Backend)
	params.Add("name", machine.Name)
	for k, v := range machine.Settings {
		params.Add("settings["+k+"]", v)
	}
	rurl += "?" + params.Encode()
	fmt.Println(rurl)

	// Setting are encoded in a bit weird way
	// Note: This is not supported by openQA at the moment, but we keep it here for when it does.
	wmach := weirdMachine{Name: machine.Name, ID: machine.ID, Backend: machine.Backend}
	wmach.CopySettingsFrom(machine)

	// Encode the machine settings as JSON
	if buf, err := i.post(rurl, wmach); err != nil {
		return Machine{}, err
	} else {
		err = json.Unmarshal(buf, &machine)
		return machine, err
	}
}
