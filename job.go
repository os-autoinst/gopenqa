package gopenqa

import "fmt"

/* Job instance */
type Job struct {
	AssignedWorkerID int      `json:"assigned_worker_id"`
	BlockedByID      int      `json:"blocked_by_id"`
	Children         Children `json:"children"`
	Parents          Children `json:"parents"`
	CloneID          int64    `json:"clone_id"`
	GroupID          int      `json:"group_id"`
	ID               int64    `json:"id"`
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
	Link     string
	Prefix   string
	Remote   string // openQA remote host
	instance *Instance
}

/* Children struct is for chained, directly chained and parallel children/parents */
type Children struct {
	Chained         []int64 `json:"Chained"`
	DirectlyChained []int64 `json:"Directly chained"`
	Parallel        []int64 `json:"Parallel"`
}

/* Job Setting struct */
type Settings struct {
	Arch    string `json:"ARCH"`
	Backend string `json:"BACKEND"`
	Machine string `json:"MACHINE"`
}

/* Format job as a string */
func (j *Job) String() string {
	return fmt.Sprintf("%d %s (%s)", j.ID, j.Name, j.Test)
}

/* JobState returns or the job state or the result, if the job is done */
func (j *Job) JobState() string {
	if j.State == "done" {
		return j.Result
	}
	return j.State
}

/* IsCloned returns true, if the job has been cloned/restarted */
func (j *Job) IsCloned() bool {
	return j.CloneID != 0 && j.CloneID != j.ID
}

/* Compares two jobs according to their unique parameters (ID, GroupID, Test) */
func (j1 *Job) Equals(j2 Job) bool {
	// Compare only relevant parameters
	if j1.ID != j2.ID {
		return false
	}
	if j1.GroupID != j2.GroupID {
		return false
	}
	if j1.Test != j2.Test {
		return false
	}
	// Consider the other parameters as volatile and thus not important

	return true
}
