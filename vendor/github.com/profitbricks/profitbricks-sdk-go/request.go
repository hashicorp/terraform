package profitbricks

import (
	"encoding/json"
	"net/http"
)

type RequestStatus struct {
	Id         string                `json:"id,omitempty"`
	Type_      string                `json:"type,omitempty"`
	Href       string                `json:"href,omitempty"`
	Metadata   RequestStatusMetadata `json:"metadata,omitempty"`
	Response   string                `json:"Response,omitempty"`
	Headers    *http.Header          `json:"headers,omitempty"`
	StatusCode int                   `json:"headers,omitempty"`
}
type RequestStatusMetadata struct {
	Status  string          `json:"status,omitempty"`
	Message string          `json:"message,omitempty"`
	Etag    string          `json:"etag,omitempty"`
	Targets []RequestTarget `json:"targets,omitempty"`
}

type RequestTarget struct {
	Target ResourceReference `json:"target,omitempty"`
	Status string            `json:"status,omitempty"`
}

func GetRequestStatus(path string) RequestStatus {
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toRequestStatus(do(req))
}

func toRequestStatus(resp Resp) RequestStatus {
	var server RequestStatus
	json.Unmarshal(resp.Body, &server)
	server.Response = string(resp.Body)
	server.Headers = &resp.Headers
	server.StatusCode = resp.StatusCode
	return server
}
