package alerts

// CreateAlertResponse holds the result data of the CreateAlertRequest
type CreateAlertResponse struct {
	Message string `json:"message"`
	AlertID string `json:"alertId"`
	Status  string `json:"status"`
	Code    int    `json:"code"`
}

// CountAlertResponse holds the result data of the CountAlertRequest
type CountAlertResponse struct {
	Count   int    `json:"count"`
}

// CloseAlertResponse holds the result data of the CloseAlertRequest
type CloseAlertResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// DeleteAlertResponse holds the result data of the DeleteAlertRequest
type DeleteAlertResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// ListAlertsResponse holds the result data of the ListAlertsRequest
type ListAlertsResponse struct {
	Alerts []struct {
		ID           string `json:"id"`
		Alias        string `json:"alias"`
		Message      string `json:"message"`
		Status       string `json:"status"`
		IsSeen       bool   `json:"isSeen"`
		Acknowledged bool   `json:"acknowledged"`
		CreatedAt    uint64 `json:"createdAt"`
		UpdatedAt    uint64 `json:"updatedAt"`
		TinyID       string `json:"tinyId"`
		Owner        string `json:"owner"`
	} `json:"alerts"`
}

// ListAlertNotesResponse holds the result data of the ListAlertNotesRequest
type ListAlertNotesResponse struct {
	Took    int    `json:"took"`
	LastKey string `json:"lastKey"`
	Notes   []struct {
		Note      string `json:"note"`
		Owner     string `json:"owner"`
		CreatedAt uint64 `json:"createdAt"`
	} `json:"notes"`
}

// ListAlertLogsResponse holds the result data of the ListAlertLogsRequest
type ListAlertLogsResponse struct {
	LastKey string `json:"lastKey"`
	Logs    []struct {
		Log       string `json:"log"`
		LogType   string `json:"logType"`
		Owner     string `json:"owner"`
		CreatedAt uint64 `json:"createdAt"`
	} `json:"logs"`
}

// ListAlertRecipientsResponse holds the result data of the ListAlertRecipientsRequest.
type ListAlertRecipientsResponse struct {
	Users []struct {
		Username       string `json:"username"`
		State          string `json:"state"`
		Method         string `json:"method"`
		StateChangedAt uint64 `json:"stateChangedAt"`
	} `json:"users"`

	Groups map[string][]struct {
		Username       string `json:"username"`
		State          string `json:"state"`
		Method         string `json:"method"`
		StateChangedAt uint64 `json:"stateChangedAt"`
	} `json:"groups"`
}

// AcknowledgeAlertResponse holds the result data of the AcknowledgeAlertRequest.
type AcknowledgeAlertResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// RenotifyAlertResponse holds the result data of the RenotifyAlertRequest.
type RenotifyAlertResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// TakeOwnershipAlertResponse holds the result data of the TakeOwnershipAlertRequest.
type TakeOwnershipAlertResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// AssignOwnerAlertResponse holds the result data of the AssignOwnerAlertRequest.
type AssignOwnerAlertResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// AddTeamAlertResponse holds the result data of the AddTeamAlertRequest.
type AddTeamAlertResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// AddRecipientAlertResponse holds the result data of the AddRecipientAlertRequest.
type AddRecipientAlertResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// AddNoteAlertResponse holds the result data of the AddNoteAlertRequest.
type AddNoteAlertResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// AddTagsAlertResponse holds the result data of the AddTagsAlertRequest.
type AddTagsAlertResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// ExecuteActionAlertResponse holds the result data of the ExecuteActionAlertRequest.
type ExecuteActionAlertResponse struct {
	Result string `json:"result"`
	Code   int    `json:"code"`
}

// AttachFileAlertResponse holds the result data of the AttachFileAlertRequest.
type AttachFileAlertResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// GetAlertResponse holds the result data of the GetAlertRequest.
type GetAlertResponse struct {
	Tags         []string               `json:"tags"`
	Count        int                    `json:"count"`
	Status       string                 `json:"status"`
	Teams        []string               `json:"teams"`
	Recipients   []string               `json:"recipients"`
	TinyID       string                 `json:"tinyId"`
	Alias        string                 `json:"alias"`
	Entity       string                 `json:"entity"`
	ID           string                 `json:"id"`
	UpdatedAt    uint64                 `json:"updatedAt"`
	Message      string                 `json:"message"`
	Details      map[string]string      `json:"details"`
	Source       string                 `json:"source"`
	Description  string                 `json:"description"`
	CreatedAt    uint64                 `json:"createdAt"`
	IsSeen       bool                   `json:"isSeen"`
	Acknowledged bool                   `json:"acknowledged"`
	Owner        string                 `json:"owner"`
	Actions      []string               `json:"actions"`
	SystemData   map[string]interface{} `json:"systemData"`
}

// UnAcknowledgeAlertResponse holds the result data of the UnAcknowledgeAlertRequest
type UnAcknowledgeAlertResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
	Took   int    `json:"took"`
}

// SnoozeAlertResponse holds the result data of the SnoozeAlertRequest
type SnoozeAlertResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// RemoveTagsAlertResponse holds the result data of the RemoveTagsAlertRequest
type RemoveTagsAlertResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// AddDetailsAlertResponse holds the result data of the AddDetailsAlertRequest
type AddDetailsAlertResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// RemoveDetailsAlertResponse holds the result data of the RemoveDetailsAlertRequest
type RemoveDetailsAlertResponse struct {
	Status	string `json:"status"`
	Code 	int    `json:"code"`
}

// EscalateToNextAlertResponse holds the result data of the EscalateToNextAlertRequest
type EscalateToNextAlertResponse struct {
	Status	string	`json:"status"`
	Code	int	`json:"code"`
}

//IntegrationType returns extracted "integrationType" data from the retrieved alert' SystemData property.
func (res *GetAlertResponse) IntegrationType() string {
	if val, ok := res.SystemData["integrationType"].(string); ok {
		return val
	}
	return ""
}

//IntegrationID returns extracted "integrationId" data from the retrieved alert' SystemData property.
func (res *GetAlertResponse) IntegrationID() string {
	if val, ok := res.SystemData["integrationId"].(string); ok {
		return val
	}
	return ""
}

//IntegrationName returns extracted "integrationName" data from the retrieved alert' SystemData property.
func (res *GetAlertResponse) IntegrationName() string {
	if val, ok := res.SystemData["integrationName"].(string); ok {
		return val
	}
	return ""
}

//AckTime returns extracted "ackTime" data from the retrieved alert' SystemData property.
func (res *GetAlertResponse) AckTime() uint64 {
	if val, ok := res.SystemData["ackTime"].(uint64); ok {
		return val
	}
	return 0
}

//AcknowledgedBy returns extracted "acknowledgedBy" data from the retrieved alert' SystemData property.
func (res *GetAlertResponse) AcknowledgedBy() string {
	if val, ok := res.SystemData["acknowledgedBy"].(string); ok {
		return val
	}
	return ""
}

//CloseTime returns extracted "closeTime" data from the retrieved alert' SystemData property.
func (res *GetAlertResponse) CloseTime() uint64 {
	if val, ok := res.SystemData["closeTime"].(uint64); ok {
		return val
	}
	return 0
}

//ClosedBy returns extracted "closedBy" data from the retrieved alert' SystemData property.
func (res *GetAlertResponse) ClosedBy() string {
	if val, ok := res.SystemData["closedBy"].(string); ok {
		return val
	}
	return ""
}
