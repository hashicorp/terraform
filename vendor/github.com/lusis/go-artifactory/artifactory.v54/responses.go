package artifactory

// GavcSearchResults represents gavc search results
type GavcSearchResults struct {
	Results []FileInfo `json:"results"`
}

// URI is a URI in artifactory json
type URI struct {
	URI string `json:"uri,omitempty"`
}
