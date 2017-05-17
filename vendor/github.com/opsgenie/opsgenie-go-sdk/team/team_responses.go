package team

// Create team response structure
type CreateTeamResponse struct {
	Id string `json:"id"`
	Status string `json:"status"`
	Code int `json:"code"`
}

// Update team response structure
type UpdateTeamResponse struct {
	Status string `json:"status"`
        Code int `json:"code"`
}

// Delete team response structure
type DeleteTeamResponse struct {
	Status string `json:"status"`
        Code int `json:"code"`
}

// Get team response structure
type GetTeamResponse struct {
	Description string `json:"description,omitempty"`
	Id string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Members []Member `json:"members,omitempty"`
}


// List teams response structure
type ListTeamsResponse struct {
	Teams []GetTeamResponse `json:"teams,omitempty"`
}

// A single team log entry
type TeamLogEntry struct {
	Log string `json:"log"`
	Owner string `json:"owner"`
	CreatedAt uint `json:"createdAt"`
}

//List team logs response structure
type ListTeamLogsResponse struct {
	LastKey string `json:"lastKey,omitempty"`
	Logs []TeamLogEntry `json:logs,omitempty`
}
