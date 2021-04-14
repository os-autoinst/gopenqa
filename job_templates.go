package gopenqa

/* Job Template */
type JobTemplate struct {
	GroupName string    `json:"group_name"`
	ID        int       `json:"id"`
	Machine   Machine   `json:"machine"`
	Priority  int       `json:"prio"`
	Product   Product   `json:"product"`
	TestSuite TestSuite `json:"test_suite"`
}
