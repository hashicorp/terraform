package group

import (
	"fmt"
	"time"

	"github.com/CenturyLinkCloud/clc-sdk/api"
	"github.com/CenturyLinkCloud/clc-sdk/status"
)

func New(client api.HTTP) *Service {
	return &Service{
		client: client,
		config: client.Config(),
	}
}

type Service struct {
	client api.HTTP
	config *api.Config
}

func (s *Service) Get(id string) (*Response, error) {
	url := fmt.Sprintf("%s/groups/%s/%s", s.config.BaseURL, s.config.Alias, id)
	resp := &Response{}
	err := s.client.Get(url, resp)
	return resp, err
}

func (s *Service) Create(group Group) (*Response, error) {
	resp := &Response{}
	url := fmt.Sprintf("%s/groups/%s", s.config.BaseURL, s.config.Alias)
	err := s.client.Post(url, group, resp)
	return resp, err
}

func (s *Service) Update(id string, updates ...api.Update) error {
	url := fmt.Sprintf("%s/groups/%s/%s", s.config.BaseURL, s.config.Alias, id)
	return s.client.Patch(url, updates, nil)
}

func (s *Service) Delete(id string) (*status.Status, error) {
	url := fmt.Sprintf("%s/groups/%s/%s", s.config.BaseURL, s.config.Alias, id)
	resp := &status.Status{}
	err := s.client.Delete(url, resp)
	return resp, err
}

func (s *Service) Archive(id string) (*status.Status, error) {
	url := fmt.Sprintf("%s/groups/%s/%s/archive", s.config.BaseURL, s.config.Alias, id)
	resp := &status.Status{}
	err := s.client.Post(url, "", resp)
	return resp, err
}

func (s *Service) Restore(id, intoGroup string) (*status.QueuedResponse, error) {
	url := fmt.Sprintf("%s/groups/%s/%s/restore", s.config.BaseURL, s.config.Alias, id)
	resp := &status.QueuedResponse{}
	body := fmt.Sprintf(`{"targetGroupId": "%v"}`, intoGroup)
	err := s.client.Post(url, body, resp)
	return resp, err
}

func (s *Service) SetDefaults(id string, defaults *GroupDefaults) error {
	url := fmt.Sprintf("%s/groups/%s/%s/defaults", s.config.BaseURL, s.config.Alias, id)
	var resp interface{}
	err := s.client.Post(url, defaults, resp)
	return err
}

func (s *Service) SetHorizontalAutoscalePolicy(id string, policy *HorizontalAutoscalePolicy) (*interface{}, error) {
	url := fmt.Sprintf("%s/groups/%s/%s/horizontalAutoscalePolicy", s.config.BaseURL, s.config.Alias, id)
	var resp interface{}
	err := s.client.Put(url, policy, resp)
	return &resp, err
}

func UpdateName(name string) api.Update {
	return api.Update{
		Op:     "set",
		Member: "name",
		Value:  name,
	}
}

func UpdateDescription(desc string) api.Update {
	return api.Update{
		Op:     "set",
		Member: "description",
		Value:  desc,
	}
}

func UpdateParentGroupID(id string) api.Update {
	return api.Update{
		Op:     "set",
		Member: "parentGroupId",
		Value:  id,
	}
}

func UpdateCustomfields(fields []api.Customfields) api.Update {
	return api.Update{
		Op:     "set",
		Member: "customFields",
		Value:  fields,
	}
}

// request body for creating groups
type Group struct {
	Name          string             `json:"name"`
	Description   string             `json:"description,omitempty"`
	ParentGroupID string             `json:"parentGroupId"`
	CustomFields  []api.Customfields `json:"customFields,omitempty"`
}

// response body for group get
type Response struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Locationid  string    `json:"locationId"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	Links       api.Links `json:"links"`
	Groups      []Groups  `json:"groups"`
	Changeinfo  struct {
		Createddate  time.Time `json:"createdDate"`
		Createdby    string    `json:"createdBy"`
		Modifieddate time.Time `json:"modifiedDate"`
		Modifiedby   string    `json:"modifiedBy"`
	} `json:"changeInfo"`
	Customfields []api.Customfields `json:"customFields"`
}

func (r *Response) ParentGroupID() string {
	if ok, link := r.Links.GetLink("parentGroup"); ok {
		return link.ID
	}
	return ""
}

func (r *Response) Servers() []string {
	ids := make([]string, 0)
	for _, l := range r.Links {
		if l.Rel == "server" {
			ids = append(ids, l.ID)
		}
	}
	return ids
}

// nested groups under response
type Groups struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Locationid   string    `json:"locationId"`
	Type         string    `json:"type"`
	Status       string    `json:"status"`
	Serverscount int       `json:"serversCount"`
	Groups       []Groups  `json:"groups"`
	Links        api.Links `json:"links"`
}

// request body for /v2/groups/ALIAS/ID/defaults
type GroupDefaults struct {
	CPU          string `json:"cpu,omitempty"`
	MemoryGB     string `json:"memoryGB,omitempty"`
	NetworkID    string `json:"networkId,omitempty"`
	primaryDns   string `json:"primaryDns,omitempty"`
	secondaryDns string `json:"secondaryDns,omitempty"`
	templateName string `json:"templateName,omitempty"`
}

// request body for /v2/groups/ALIAS/ID/horizontalAutoscalePolicy
type HorizontalAutoscalePolicy struct {
	PolicyId         string       `json:"policyId,omitempty"`
	LoadBalancerPool []PoolPolicy `json:"loadBalancerPool,omitempty"`
}

type PoolPolicy struct {
	ID          string `json:"id,omitempty"`
	PrivatePort int    `json:"privatePort,omitempty"`
	PublicPort  int    `json:"publicPort,omitempty"`
}
