package artifactory

type ErrorsJson struct {
	Errors []ErrorJson `json:"errors"`
}

type ErrorJson struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
