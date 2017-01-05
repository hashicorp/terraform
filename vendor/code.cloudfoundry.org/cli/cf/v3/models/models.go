package models

type V3Application struct {
	Name                  string `json:"name"`
	DesiredState          string `json:"desired_state"`
	TotalDesiredInstances int    `json:"total_desired_instances"`
	Links                 Links  `json:"links"`
}

type Links struct {
	Processes Link `json:"processes"`
	Routes    Link `json:"routes"`
}

type Link struct {
	Href string `json:"href"`
}

type V3Process struct {
	Type       string `json:"type"`
	Instances  int    `json:"instances"`
	MemoryInMB int64  `json:"memory_in_mb"`
	DiskInMB   int64  `json:"disk_in_mb"`
}

type V3Route struct {
	Host string `json:"host"`
	Path string `json:"path"`
}
