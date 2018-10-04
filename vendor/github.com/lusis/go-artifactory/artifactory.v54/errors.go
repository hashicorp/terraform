package artifactory

// ErrorsJSON represents a group of error messages returned by the artifactory api
type ErrorsJSON struct {
	Errors []ErrorJSON `json:"errors,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// ErrorJSON represents a single error message returned by the artifactory API
type ErrorJSON struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
