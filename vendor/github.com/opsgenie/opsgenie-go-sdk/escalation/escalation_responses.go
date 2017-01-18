package escalation

// Create escalation response structure
type CreateEscalationResponse struct {
	Id     string `json:"id"`
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// Update escalation response structure
type UpdateEscalationResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// Delete escalation response structure
type DeleteEscalationResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// Get escalation structure
type GetEscalationResponse struct {
	Id    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Team  string `json:"team,omitempty"`
	Rules []Rule `json:"rules,omitempty"`
}

// List escalations response structure
type ListEscalationsResponse struct {
	Escalations []GetEscalationResponse `json:"escalations,omitempty"`
}
