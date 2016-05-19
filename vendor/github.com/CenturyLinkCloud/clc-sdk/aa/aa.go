package aa

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

func (s *Service) Get(id string) (*Policy, error) {
	url := fmt.Sprintf("%s/antiAffinityPolicies/%s/%s", s.config.BaseURL, s.config.Alias, id)
	policy := &Policy{}
	err := s.client.Get(url, policy)
	return policy, err
}

func (s *Service) GetAll() (*Policies, error) {
	url := fmt.Sprintf("%s/antiAffinityPolicies/%s", s.config.BaseURL, s.config.Alias)
	policies := &Policies{}
	err := s.client.Get(url, policies)
	return policies, err
}

func (s *Service) Create(name, location string) (*Policy, error) {
	policy := &Policy{Name: name, Location: location}
	resp := &Policy{}
	url := fmt.Sprintf("%s/antiAffinityPolicies/%s", s.config.BaseURL, s.config.Alias)
	err := s.client.Post(url, policy, resp)
	return resp, err
}

func (s *Service) Update(id string, name string) (*Policy, error) {
	policy := &Policy{Name: name}
	resp := &Policy{}
	url := fmt.Sprintf("%s/antiAffinityPolicies/%s/%s", s.config.BaseURL, s.config.Alias, id)
	err := s.client.Put(url, policy, resp)
	return resp, err
}

func (s *Service) Delete(id string) error {
	url := fmt.Sprintf("%s/antiAffinityPolicies/%s/%s", s.config.BaseURL, s.config.Alias, id)
	err := s.client.Delete(url, nil)
	return err
}

type Policy struct {
	ID       string    `json:"id,omitempty"`
	Name     string    `json:"name,omitempty"`
	Location string    `json:"location,omitempty"`
	Links    api.Links `json:"links,omitempty"`
}

type Policies struct {
	Items []Policy  `json:"items,omitempty"`
	Links api.Links `json:"links,omitempty"`
}
