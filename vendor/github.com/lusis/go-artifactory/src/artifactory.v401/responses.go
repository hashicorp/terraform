package artifactory

type GavcSearchResults struct {
	Results []FileInfo `json:"results"`
}

type Uri struct {
	Uri string `json:"uri,omitempty"`
}
