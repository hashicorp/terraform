package nsone

type FeedDestination struct {
	Destid   string `json"destif"`
	Desttype string `json"desttype"`
	Record   string `json"record"`
}

type DataSource struct {
	Id           string            `json:"id,omitempty"`
	Name         string            `json:"name"`
	SourceType   string            `json:"sourcetype"`
	Config       map[string]string `json:"config,omitempty"`
	Status       string            `json:"status,omitempty"`
	Destinations []FeedDestination `json:"destinations,omitempty"`
}

func NewDataSource(name string, source_type string) *DataSource {
	cf := make(map[string]string, 0)
	return &DataSource{
		Name:       name,
		SourceType: source_type,
		Config:     cf,
	}
}
