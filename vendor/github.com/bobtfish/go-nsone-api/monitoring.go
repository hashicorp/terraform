package nsone

type MonitoringJobTypes map[string]MonitoringJobType
type MonitoringJobType struct {
	ShortDesc string                   `json:"shortdesc"`
	Config    MonitoringJobTypeConfig  `json:"config"`
	Results   MonitoringJobTypeResults `json:"results"`
	Desc      string                   `json:"desc"`
}
type MonitoringJobTypeConfig map[string]interface{}
type MonitoringJobTypeResults map[string]MonitoringJobTypeResult
type MonitoringJobTypeResult struct {
	Comparators []string `json:"comparators"`
	Metric      bool     `json:"metric"`
	Validator   string   `json:"validator"`
	ShortDesc   string   `json:"shortdesc"`
	Type        string   `json:type"`
	Desc        string   `json:"desc"`
}

type MonitoringJobs []MonitoringJob
type MonitoringJob struct {
	Id             string                         `json:"id,omitempty"`
	Config         map[string]interface{}         `json:"config"`
	Status         map[string]MonitoringJobStatus `json:"status,omitempty"`
	Rules          []MonitoringJobRule            `json:"rules"`
	JobType        string                         `json:"job_type"`
	Regions        []string                       `json:"regions"`
	Active         bool                           `json:"active"`
	Frequency      int                            `json:"frequency"`
	Policy         string                         `json:"policy"`
	RegionScope    string                         `json:"region_scope"`
	Notes          string                         `json:"notes,omitempty"`
	Name           string                         `json:"name"`
	NotifyRepeat   int                            `json:"notify_repeat"`
	RapidRecheck   bool                           `json:"rapid_recheck"`
	NotifyDelay    int                            `json:"notify_delay"`
	NotifyList     string                         `json:"notify_list"`
	NotifyRegional bool                           `json:"notidy_regional"`
	NotifyFailback bool                           `json:"notify_failback"`
}
type MonitoringJobStatus struct {
	Since  int    `json:"since"`
	Status string `json:"status"`
}
type MonitoringJobRule struct {
	Key        string      `json:"key"`
	Value      interface{} `json:"value"`
	Comparison string      `json:"comparison"`
}
