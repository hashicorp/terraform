package profitbricks

import (
	"encoding/json"
	"net/http"
	"time"
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

type Requests struct {
	Id         string       `json:"id,omitempty"`
	Type_      string       `json:"type,omitempty"`
	Href       string       `json:"href,omitempty"`
	Items      []Request     `json:"items,omitempty"`
	Response   string       `json:"Response,omitempty"`
	Headers    *http.Header `json:"headers,omitempty"`
	StatusCode int          `json:"headers,omitempty"`
}

type Request struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Href       string `json:"href"`
	Metadata   struct {
			   CreatedDate   time.Time `json:"createdDate"`
			   CreatedBy     string `json:"createdBy"`
			   Etag          string `json:"etag"`
			   RequestStatus struct {
						 ID   string `json:"id"`
						 Type string `json:"type"`
						 Href string `json:"href"`
					 } `json:"requestStatus"`
		   } `json:"metadata"`
	Properties struct {
			   Method  string `json:"method"`
			   Headers interface{} `json:"headers"`
			   Body    interface{} `json:"body"`
			   URL     string `json:"url"`
		   } `json:"properties"`
	Response   string       `json:"Response,omitempty"`
	Headers    *http.Header `json:"headers,omitempty"`
	StatusCode int          `json:"headers,omitempty"`
}

func ListRequests() Requests {
	url := mk_url("/requests") + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toRequests(do(req))
}

func GetRequest(req_id string) Request {
	url := mk_url("/requests/" + req_id) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toRequest(do(req))
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

func toRequests(resp Resp) Requests {
	var server Requests
	json.Unmarshal(resp.Body, &server)
	server.Response = string(resp.Body)
	server.Headers = &resp.Headers
	server.StatusCode = resp.StatusCode
	return server
}

func toRequest(resp Resp) Request {
	var server Request
	json.Unmarshal(resp.Body, &server)
	server.Response = string(resp.Body)
	server.Headers = &resp.Headers
	server.StatusCode = resp.StatusCode
	return server
}

