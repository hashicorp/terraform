// Copyright 2016 go-dockerclient authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package docker

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/docker/docker/api/types/swarm"
	"golang.org/x/net/context"
)

// NoSuchService is the error returned when a given service does not exist.
type NoSuchService struct {
	ID  string
	Err error
}

func (err *NoSuchService) Error() string {
	if err.Err != nil {
		return err.Err.Error()
	}
	return "No such service: " + err.ID
}

// CreateServiceOptions specify parameters to the CreateService function.
//
// See https://goo.gl/KrVjHz for more details.
type CreateServiceOptions struct {
	swarm.ServiceSpec
	Context context.Context
}

// CreateService creates a new service, returning the service instance
// or an error in case of failure.
//
// See https://goo.gl/KrVjHz for more details.
func (c *Client) CreateService(opts CreateServiceOptions) (*swarm.Service, error) {
	path := "/services/create?" + queryString(opts)
	resp, err := c.do("POST", path, doOptions{
		data:      opts.ServiceSpec,
		forceJSON: true,
		context:   opts.Context,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var service swarm.Service
	if err := json.NewDecoder(resp.Body).Decode(&service); err != nil {
		return nil, err
	}
	return &service, nil
}

// RemoveServiceOptions encapsulates options to remove a service.
//
// See https://goo.gl/Tqrtya for more details.
type RemoveServiceOptions struct {
	ID      string `qs:"-"`
	Context context.Context
}

// RemoveService removes a service, returning an error in case of failure.
//
// See https://goo.gl/Tqrtya for more details.
func (c *Client) RemoveService(opts RemoveServiceOptions) error {
	path := "/services/" + opts.ID
	resp, err := c.do("DELETE", path, doOptions{context: opts.Context})
	if err != nil {
		if e, ok := err.(*Error); ok && e.Status == http.StatusNotFound {
			return &NoSuchService{ID: opts.ID}
		}
		return err
	}
	resp.Body.Close()
	return nil
}

// UpdateServiceOptions specify parameters to the UpdateService function.
//
// See https://goo.gl/wu3MmS for more details.
type UpdateServiceOptions struct {
	swarm.ServiceSpec
	Context context.Context
	Version uint64
}

// UpdateService updates the service at ID with the options
//
// See https://goo.gl/wu3MmS for more details.
func (c *Client) UpdateService(id string, opts UpdateServiceOptions) error {
	params := make(url.Values)
	params.Set("version", strconv.FormatUint(opts.Version, 10))
	resp, err := c.do("POST", "/services/"+id+"/update?"+params.Encode(), doOptions{
		data:      opts.ServiceSpec,
		forceJSON: true,
		context:   opts.Context,
	})
	if err != nil {
		if e, ok := err.(*Error); ok && e.Status == http.StatusNotFound {
			return &NoSuchService{ID: id}
		}
		return err
	}
	defer resp.Body.Close()
	return nil
}

// InspectService returns information about a service by its ID.
//
// See https://goo.gl/dHmr75 for more details.
func (c *Client) InspectService(id string) (*swarm.Service, error) {
	path := "/services/" + id
	resp, err := c.do("GET", path, doOptions{})
	if err != nil {
		if e, ok := err.(*Error); ok && e.Status == http.StatusNotFound {
			return nil, &NoSuchService{ID: id}
		}
		return nil, err
	}
	defer resp.Body.Close()
	var service swarm.Service
	if err := json.NewDecoder(resp.Body).Decode(&service); err != nil {
		return nil, err
	}
	return &service, nil
}

// ListServicesOptions specify parameters to the ListServices function.
//
// See https://goo.gl/DwvNMd for more details.
type ListServicesOptions struct {
	Filters map[string][]string
	Context context.Context
}

// ListServices returns a slice of services matching the given criteria.
//
// See https://goo.gl/DwvNMd for more details.
func (c *Client) ListServices(opts ListServicesOptions) ([]swarm.Service, error) {
	path := "/services?" + queryString(opts)
	resp, err := c.do("GET", path, doOptions{context: opts.Context})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var services []swarm.Service
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return nil, err
	}
	return services, nil
}
