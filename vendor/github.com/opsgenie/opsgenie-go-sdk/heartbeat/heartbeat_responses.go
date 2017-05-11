package heartbeat

// AddHeartbeatResponse holds the result data of the AddHeartbeatRequest.
type AddHeartbeatResponse struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// UpdateHeartbeatResponse holds the result data of the UpdateHeartbeatRequest.
type UpdateHeartbeatResponse struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// EnableHeartbeatResponse holds the result data of the EnableHeartbeatRequest.
type EnableHeartbeatResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// DisableHeartbeatResponse holds the result data of the DisableHeartbeatRequest.
type DisableHeartbeatResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// DeleteHeartbeatResponse holds the result data of the DeleteHeartbeatRequest.
type DeleteHeartbeatResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// GetHeartbeatResponse holds the result data of the GetHeartbeatRequest.
type GetHeartbeatResponse struct {
	Heartbeat
}

// ListHeartbeatsResponse holds the result data of the ListHeartbeatsRequest.
type ListHeartbeatsResponse struct {
	Heartbeats []Heartbeat `json:"heartbeats"`
}

type Heartbeat struct {
	Name          string `json:"name"`
	Status        string `json:"status"`
	Description   string `json:"description"`
	Enabled       bool   `json:"enabled"`
	LastHeartbeat uint64 `json:"lastHeartBeat"`
	Interval      int    `json:"interval"`
	IntervalUnit  string `json:"intervalUnit"`
	Expired       bool   `json:"expired"`
}

// SendHeartbeatResponse holds the result data of the SendHeartbeatRequest.
type SendHeartbeatResponse struct {
	WillExpireAt uint64 `json:"willExpireAt"`
	Status       string `json:"status"`
	Heartbeat    uint64 `json:"heartbeat"`
	Took         int    `json:"took"`
	Code         int    `json:"code"`
}
