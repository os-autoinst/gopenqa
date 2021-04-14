package gopenqa

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
	Link     string
	Prefix   string
	instance *Instance
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
