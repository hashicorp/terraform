package integration

// EnableIntegrationResponse holds the result data of the EnableIntegrationRequest.
type EnableIntegrationResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// DisableIntegrationResponse holds the result data of the DisableIntegrationRequest.
type DisableIntegrationResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}
