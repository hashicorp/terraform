package alert

import (
	"fmt"

	"github.com/CenturyLinkCloud/clc-sdk/api"
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

func (s *Service) Get(id string) (*Alert, error) {
	url := fmt.Sprintf("%s/alertPolicies/%s/%s", s.config.BaseURL, s.config.Alias, id)
	resp := &Alert{}
	err := s.client.Get(url, resp)
	return resp, err
}

func (s *Service) GetAll() (*Alerts, error) {
	url := fmt.Sprintf("%s/alertPolicies/%s", s.config.BaseURL, s.config.Alias)
	resp := &Alerts{}
	err := s.client.Get(url, resp)
	return resp, err
}

func (s *Service) Create(alert Alert) (*Alert, error) {
	url := fmt.Sprintf("%s/alertPolicies/%s", s.config.BaseURL, s.config.Alias)
	resp := &Alert{}
	err := s.client.Post(url, alert, resp)
	return resp, err
}

func (s *Service) Update(id string, alert Alert) (*Alert, error) {
	url := fmt.Sprintf("%s/alertPolicies/%s/%s", s.config.BaseURL, s.config.Alias, id)
	resp := &Alert{}
	err := s.client.Put(url, alert, resp)
	return resp, err
}

func (s *Service) Delete(id string) error {
	url := fmt.Sprintf("%s/alertPolicies/%s/%s", s.config.BaseURL, s.config.Alias, id)
	return s.client.Delete(url, nil)
}

type Alerts struct {
	Items []Alert   `json:"items"`
	Links api.Links `json:"links"`
}

type Alert struct {
	ID       string    `json:"id,omitempty"`
	Name     string    `json:"name,omitempty"`
	Actions  []Action  `json:"actions,omitempty"`
	Triggers []Trigger `json:"triggers,omitempty"`
	Links    api.Links `json:"links,omitempty"`
}

type Action struct {
	Action  string  `json:"action"`
	Setting Setting `json:"settings"`
}

type Setting struct {
	Recipients []string `json:"recipients"`
}

type Trigger struct {
	Metric    string  `json:"metric"`
	Duration  string  `json:"duration"`
	Threshold float64 `json:"threshold"`
}
