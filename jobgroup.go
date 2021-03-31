package gopenqa

import (
	"fmt"
	"net/url"
)

/* Job Group */
type JobGroup struct {
	ID                         int    `json:"id"`
	Name                       string `json:"name"`
	ParentID                   int    `json:"parent_id"`
	Description                string `json:"description"`
	BuildVersionSort           int    `json:"build_version_sort"`
	CarryOverBugrefs           int    `json:"carry_over_bugrefs"`
	DefaultPriority            int    `json:"default_priority`
	KeepImportantLogsInDays    int    `json:"keep_important_logs_in_days"`
	KeepImportantResultsInDays int    `json:"keep_important_results_in_days"`
	KeepLogsInDays             int    `json:"keep_logs_in_days"`
	KeepResultsInDays          int    `json:"keep_results_in_days"`
	SizeLimit                  int    `json:"size_limit_gb"` // Size limit in GB
	SortOrder                  int    `json:"sort_order"`
	Template                   string `json:"template"`
}

func addIntIfNotZero(value int, name string, values *url.Values) {
	if value > 0 {
		values.Add(name, fmt.Sprintf("%d", value))
	}
}

/* Get www-form-urlencoded parameters of this Product */
func (j *JobGroup) encodeWWW() string {
	params := url.Values{}
	addIntIfNotZero(j.ID, "id", &params)
	params.Add("name", j.Name)
	if j.ParentID > 0 {
		params.Add("parent_id", fmt.Sprintf("%d", j.ParentID))
	}
	params.Add("description", j.Description)

	addIntIfNotZero(j.BuildVersionSort, "build_version_sort", &params)
	addIntIfNotZero(j.CarryOverBugrefs, "carry_over_bugrefs", &params)
	addIntIfNotZero(j.DefaultPriority, "default_priority", &params)
	addIntIfNotZero(j.KeepImportantLogsInDays, "keep_important_logs_in_days", &params)
	addIntIfNotZero(j.KeepImportantResultsInDays, "keep_important_results_in_days", &params)
	addIntIfNotZero(j.KeepLogsInDays, "keep_logs_in_days", &params)
	addIntIfNotZero(j.KeepResultsInDays, "keep_results_in_days", &params)
	addIntIfNotZero(j.SizeLimit, "size_limit_gb", &params)
	addIntIfNotZero(j.SortOrder, "sort_order", &params)
	params.Add("template", j.Template)

	return params.Encode()
}
