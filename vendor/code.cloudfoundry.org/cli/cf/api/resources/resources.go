package resources

type Metadata struct {
	GUID string `json:"guid"`
	URL  string `json:"url,omitempty"`
}

type Resource struct {
	Metadata Metadata
}
