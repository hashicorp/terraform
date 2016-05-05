package lb

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

func (s *Service) Get(dc, id string) (*LoadBalancer, error) {
	url := fmt.Sprintf("%s/sharedLoadBalancers/%s/%s/%s", s.config.BaseURL, s.config.Alias, dc, id)
	resp := &LoadBalancer{}
	err := s.client.Get(url, resp)
	return resp, err
}

func (s *Service) GetAll(dc string) ([]*LoadBalancer, error) {
	url := fmt.Sprintf("%s/sharedLoadBalancers/%s/%s", s.config.BaseURL, s.config.Alias, dc)
	resp := make([]*LoadBalancer, 0)
	err := s.client.Get(url, &resp)
	return resp, err
}

func (s *Service) Create(dc string, lb LoadBalancer) (*LoadBalancer, error) {
	url := fmt.Sprintf("%s/sharedLoadBalancers/%s/%s", s.config.BaseURL, s.config.Alias, dc)
	resp := &LoadBalancer{}
	err := s.client.Post(url, lb, resp)
	return resp, err
}

func (s *Service) Update(dc, id string, lb LoadBalancer) error {
	url := fmt.Sprintf("%s/sharedLoadBalancers/%s/%s/%s", s.config.BaseURL, s.config.Alias, dc, id)
	err := s.client.Put(url, lb, nil)
	return err
}

func (s *Service) Delete(dc, id string) error {
	url := fmt.Sprintf("%s/sharedLoadBalancers/%s/%s/%s", s.config.BaseURL, s.config.Alias, dc, id)
	return s.client.Delete(url, nil)
}

func (s *Service) GetPool(dc, lb, pool string) (*Pool, error) {
	url := fmt.Sprintf("%s/sharedLoadBalancers/%s/%s/%s/pools/%s", s.config.BaseURL, s.config.Alias, dc, lb, pool)
	resp := &Pool{}
	err := s.client.Get(url, resp)
	return resp, err
}

func (s *Service) GetAllPools(dc, lb string) ([]*Pool, error) {
	url := fmt.Sprintf("%s/sharedLoadBalancers/%s/%s/%s/pools", s.config.BaseURL, s.config.Alias, dc, lb)
	resp := make([]*Pool, 0)
	err := s.client.Get(url, &resp)
	return resp, err
}

func (s *Service) CreatePool(dc, lb string, pool Pool) (*Pool, error) {
	url := fmt.Sprintf("%s/sharedLoadBalancers/%s/%s/%s/pools", s.config.BaseURL, s.config.Alias, dc, lb)
	resp := &Pool{}
	err := s.client.Post(url, pool, resp)
	return resp, err
}

func (s *Service) UpdatePool(dc, lb, id string, pool Pool) error {
	url := fmt.Sprintf("%s/sharedLoadBalancers/%s/%s/%s/pools/%s", s.config.BaseURL, s.config.Alias, dc, lb, id)
	err := s.client.Put(url, pool, nil)
	return err
}

func (s *Service) DeletePool(dc, lb, pool string) error {
	url := fmt.Sprintf("%s/sharedLoadBalancers/%s/%s/%s/pools/%s", s.config.BaseURL, s.config.Alias, dc, lb, pool)
	return s.client.Delete(url, nil)
}

func (s *Service) GetAllNodes(dc, lb, pool string) ([]*Node, error) {
	url := fmt.Sprintf("%s/sharedLoadBalancers/%s/%s/%s/pools/%s/nodes", s.config.BaseURL, s.config.Alias, dc, lb, pool)
	resp := make([]*Node, 0)
	err := s.client.Get(url, &resp)
	return resp, err
}

func (s *Service) UpdateNodes(dc, lb, pool string, nodes ...Node) error {
	url := fmt.Sprintf("%s/sharedLoadBalancers/%s/%s/%s/pools/%s/nodes", s.config.BaseURL, s.config.Alias, dc, lb, pool)
	err := s.client.Put(url, nodes, nil)
	return err
}

type LoadBalancer struct {
	ID          string    `json:"id,omitempty"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	IPaddress   string    `json:"ipAddress,omitempty"`
	Status      string    `json:"status,omitempty"`
	Pools       []Pool    `json:"pools,omitempty"`
	Links       api.Links `json:"links,omitempty"`
}

type Pool struct {
	ID          string      `json:"id,omitempty"`
	Port        int         `json:"port,omitempty"`
	Method      Method      `json:"method"`
	Persistence Persistence `json:"persistence"`
	Nodes       []Node      `json:"nodes,omitempty"`
	Links       api.Links   `json:"links,omitempty"`
}

type Node struct {
	Status      string `json:"status,omitempty"`
	IPaddress   string `json:"ipAddress"`
	PrivatePort int    `json:"privatePort"`
}

type Persistence string

const (
	Standard Persistence = "standard"
	Sticky   Persistence = "sticky"
)

type Method string

const (
	LeastConn  Method = "leastConnection"
	RoundRobin Method = "roundRobin"
)
