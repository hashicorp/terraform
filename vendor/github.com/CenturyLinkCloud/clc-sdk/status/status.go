package status

import (
	"fmt"
	"net/url"
	"time"

	"github.com/CenturyLinkCloud/clc-sdk/api"
)

func New(client api.HTTP) *Service {
	return &Service{
		client:       client,
		config:       client.Config(),
		PollInterval: 30 * time.Second,
	}
}

type Service struct {
	client api.HTTP
	config *api.Config

	PollInterval time.Duration
}

func (s *Service) Get(id string) (*Response, error) {
	url := fmt.Sprintf("%s/operations/%s/status/%s", s.config.BaseURL, s.config.Alias, id)
	status := &Response{}
	err := s.client.Get(url, status)
	return status, err
}

func (s *Service) Poll(id string, poll chan *Response) error {
	for {
		status, err := s.Get(id)
		if err != nil {
			return err
		}

		if !status.Running() {
			poll <- status
			return nil
		}
		time.Sleep(s.PollInterval)
	}
}

type Status struct {
	ID   string `json:"id"`
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

/*
   Response represents a running async job
   result from polling status
   {"status": "succeeded"}
*/
type Response struct {
	Status string `json:"status"`
}

func (s *Response) Complete() bool {
	return s.Status == Complete
}

func (s *Response) Failed() bool {
	return s.Status == Failed
}

func (s *Response) Running() bool {
	return !s.Complete() && !s.Failed() && s.Status != ""
}

const (
	Complete = "succeeded"
	Failed   = "failed"
)

/* QueuedResponse represents a returned response for an async platform job
   eg. create server
   {"server":"web", "isQueued":true, "links":[
     {"rel":"status", "href":"...", "id":"wa1-12345"},
     {"rel":"self",  "href":"...", "id":"8134c91a66784c6dada651eba90a5123"}]}
*/
type QueuedResponse struct {
	Server   string    `json:"server,omitempty"`
	IsQueued bool      `json:"isQueued,omitempty"`
	Links    api.Links `json:"links,omitempty"`
	Error    string    `json:"errorMessage,omitempty"`
}

func (q *QueuedResponse) GetStatusID() (bool, string) {
	return q.Links.GetID("status")
}

/* QueuedOperation may be a one-off and/or experimental version of QueuedResponse
   eg. add secondary network
   {"operationId": "2b70710dba4142dcaf3ab2de68e4f40c", "uri": "..."}
*/
type QueuedOperation struct {
	OperationID string `json:"operationId,omitempty"`
	URI         string `json:"uri,omitempty"`
}

func (q *QueuedOperation) GetStatusID() (bool, string) {
	return q.OperationID != "", q.OperationID
}

func (q *QueuedOperation) GetHref() (bool, string) {
	var path = ""
	if q.URI != "" {
		u, err := url.Parse(q.URI)
		if err == nil {
			path = u.Path
		}
	}
	return path != "", path
}

func (q *QueuedOperation) Status() *Status {
	st := &Status{}
	if ok, id := q.GetStatusID(); ok {
		st.ID = id
	}
	if ok, href := q.GetHref(); ok {
		st.Href = href
	}
	return st
}
