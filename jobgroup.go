package gopenqa

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
