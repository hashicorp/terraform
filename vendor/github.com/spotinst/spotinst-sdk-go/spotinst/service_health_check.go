package spotinst

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/spotinst/spotinst-sdk-go/spotinst/util/uritemplates"
)

// HealthCheck is an interface for interfacing with the HealthCheck
// endpoints of the Spotinst API.
type HealthCheckService interface {
	List(*ListHealthCheckInput) (*ListHealthCheckOutput, error)
	Create(*CreateHealthCheckInput) (*CreateHealthCheckOutput, error)
	Read(*ReadHealthCheckInput) (*ReadHealthCheckOutput, error)
	Update(*UpdateHealthCheckInput) (*UpdateHealthCheckOutput, error)
	Delete(*DeleteHealthCheckInput) (*DeleteHealthCheckOutput, error)
}

// HealthCheckServiceOp handles communication with the balancer related methods
// of the Spotinst API.
type HealthCheckServiceOp struct {
	client *Client
}

var _ HealthCheckService = &HealthCheckServiceOp{}

type HealthCheck struct {
	ID         *string            `json:"id,omitempty"`
	Name       *string            `json:"name,omitempty"`
	ResourceID *string            `json:"resourceId,omitempty"`
	Check      *HealthCheckConfig `json:"check,omitempty"`
	*HealthCheckProxy
}

type HealthCheckProxy struct {
	Addr *string `json:"proxyAddress,omitempty"`
	Port *int    `json:"proxyPort,omitempty"`
}

type HealthCheckConfig struct {
	Protocol *string `json:"protocol,omitempty"`
	Endpoint *string `json:"endpoint,omitempty"`
	Port     *int    `json:"port,omitempty"`
	Interval *int    `json:"interval,omitempty"`
	Timeout  *int    `json:"timeout,omitempty"`
	*HealthCheckThreshold
}

type HealthCheckThreshold struct {
	Healthy   *int `json:"healthyThreshold,omitempty"`
	Unhealthy *int `json:"unhealthyThreshold,omitempty"`
}

type ListHealthCheckInput struct{}

type ListHealthCheckOutput struct {
	HealthChecks []*HealthCheck `json:"healthChecks,omitempty"`
}

type CreateHealthCheckInput struct {
	HealthCheck *HealthCheck `json:"healthCheck,omitempty"`
}

type CreateHealthCheckOutput struct {
	HealthCheck *HealthCheck `json:"healthCheck,omitempty"`
}

type ReadHealthCheckInput struct {
	ID *string `json:"healthCheckId,omitempty"`
}

type ReadHealthCheckOutput struct {
	HealthCheck *HealthCheck `json:"healthCheck,omitempty"`
}

type UpdateHealthCheckInput struct {
	HealthCheck *HealthCheck `json:"healthCheck,omitempty"`
}

type UpdateHealthCheckOutput struct {
	HealthCheck *HealthCheck `json:"healthCheck,omitempty"`
}

type DeleteHealthCheckInput struct {
	ID *string `json:"healthCheckId,omitempty"`
}

type DeleteHealthCheckOutput struct{}

func healthCheckFromJSON(in []byte) (*HealthCheck, error) {
	b := new(HealthCheck)
	if err := json.Unmarshal(in, b); err != nil {
		return nil, err
	}
	return b, nil
}

func healthChecksFromJSON(in []byte) ([]*HealthCheck, error) {
	var rw responseWrapper
	if err := json.Unmarshal(in, &rw); err != nil {
		return nil, err
	}
	out := make([]*HealthCheck, len(rw.Response.Items))
	if len(out) == 0 {
		return out, nil
	}
	for i, rb := range rw.Response.Items {
		b, err := healthCheckFromJSON(rb)
		if err != nil {
			return nil, err
		}
		out[i] = b
	}
	return out, nil
}

func healthChecksFromHttpResponse(resp *http.Response) ([]*HealthCheck, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return healthChecksFromJSON(body)
}

func (s *HealthCheckServiceOp) List(input *ListHealthCheckInput) (*ListHealthCheckOutput, error) {
	r := s.client.newRequest("GET", "/healthCheck")

	_, resp, err := requireOK(s.client.doRequest(r))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	hcs, err := healthChecksFromHttpResponse(resp)
	if err != nil {
		return nil, err
	}

	return &ListHealthCheckOutput{HealthChecks: hcs}, nil
}

func (s *HealthCheckServiceOp) Create(input *CreateHealthCheckInput) (*CreateHealthCheckOutput, error) {
	r := s.client.newRequest("POST", "/healthCheck")
	r.obj = input

	_, resp, err := requireOK(s.client.doRequest(r))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	hcs, err := healthChecksFromHttpResponse(resp)
	if err != nil {
		return nil, err
	}

	output := new(CreateHealthCheckOutput)
	if len(hcs) > 0 {
		output.HealthCheck = hcs[0]
	}

	return output, nil
}

func (s *HealthCheckServiceOp) Read(input *ReadHealthCheckInput) (*ReadHealthCheckOutput, error) {
	path, err := uritemplates.Expand("/healthCheck/{healthCheckId}", map[string]string{
		"healthCheckId": StringValue(input.ID),
	})
	if err != nil {
		return nil, err
	}

	r := s.client.newRequest("GET", path)
	r.obj = input

	_, resp, err := requireOK(s.client.doRequest(r))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	hcs, err := healthChecksFromHttpResponse(resp)
	if err != nil {
		return nil, err
	}

	output := new(ReadHealthCheckOutput)
	if len(hcs) > 0 {
		output.HealthCheck = hcs[0]
	}

	return output, nil
}

func (s *HealthCheckServiceOp) Update(input *UpdateHealthCheckInput) (*UpdateHealthCheckOutput, error) {
	path, err := uritemplates.Expand("/healthCheck/{healthCheckId}", map[string]string{
		"healthCheckId": StringValue(input.HealthCheck.ID),
	})
	if err != nil {
		return nil, err
	}

	// We do not need the ID anymore so let's drop it.
	input.HealthCheck.ID = nil

	r := s.client.newRequest("PUT", path)
	r.obj = input

	_, resp, err := requireOK(s.client.doRequest(r))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	hcs, err := healthChecksFromHttpResponse(resp)
	if err != nil {
		return nil, err
	}

	output := new(UpdateHealthCheckOutput)
	if len(hcs) > 0 {
		output.HealthCheck = hcs[0]
	}

	return output, nil
}

func (s *HealthCheckServiceOp) Delete(input *DeleteHealthCheckInput) (*DeleteHealthCheckOutput, error) {
	path, err := uritemplates.Expand("/healthCheck/{healthCheckId}", map[string]string{
		"healthCheckId": StringValue(input.ID),
	})
	if err != nil {
		return nil, err
	}

	r := s.client.newRequest("DELETE", path)
	r.obj = input

	_, resp, err := requireOK(s.client.doRequest(r))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &DeleteHealthCheckOutput{}, nil
}
