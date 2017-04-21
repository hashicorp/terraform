/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	types "github.com/ukcloud/govcloudair/types/v56"
)

// Client provides a client to vCloud Air, values can be populated automatically using the Authenticate method.
type VAClient struct {
	VAToken    string  // vCloud Air authorization token
	VAEndpoint url.URL // vCloud Air API endpoint
	Region     string  // Region where the compute resource lives.
	Client     Client  // Client for the underlying vCD instance
}

// VCHS API

type services struct {
	Service []struct {
		Region      string `xml:"region,attr"`
		ServiceID   string `xml:"serviceId,attr"`
		ServiceType string `xml:"serviceType,attr"`
		Type        string `xml:"type,attr"`
		HREF        string `xml:"href,attr"`
	} `xml:"Service"`
}

type session struct {
	Link []*types.Link `xml:"Link"`
}

type computeResources struct {
	VdcRef []struct {
		Status string        `xml:"status,attr"`
		Name   string        `xml:"name,attr"`
		Type   string        `xml:"type,attr"`
		HREF   string        `xml:"href,attr"`
		Link   []*types.Link `xml:"Link"`
	} `xml:"VdcRef"`
}

type vCloudSession struct {
	VdcLink []struct {
		AuthorizationToken  string `xml:"authorizationToken,attr"`
		AuthorizationHeader string `xml:"authorizationHeader,attr"`
		Name                string `xml:"name,attr"`
		HREF                string `xml:"href,attr"`
	} `xml:"VdcLink"`
}

//

func (c *VAClient) vaauthorize(user, pass string) (u url.URL, err error) {

	if user == "" {
		user = os.Getenv("VCLOUDAIR_USERNAME")
	}

	if pass == "" {
		pass = os.Getenv("VCLOUDAIR_PASSWORD")
	}

	s := c.VAEndpoint
	s.Path += "/vchs/sessions"

	// No point in checking for errors here
	req := c.Client.NewRequest(map[string]string{}, "POST", s, nil)

	// Set Basic Authentication Header
	req.SetBasicAuth(user, pass)

	// Add the Accept header for vCA
	req.Header.Add("Accept", "application/xml;version=5.6")

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return url.URL{}, err
	}
	defer resp.Body.Close()

	// Store the authentication header
	c.VAToken = resp.Header.Get("X-Vchs-Authorization")

	session := new(session)

	if err = decodeBody(resp, session); err != nil {
		return url.URL{}, fmt.Errorf("error decoding session response: %s", err)
	}

	// Loop in the session struct to find right service and compute resource.
	for _, s := range session.Link {
		if s.Type == "application/xml;class=vnd.vmware.vchs.servicelist" && s.Rel == "down" {
			u, err := url.ParseRequestURI(s.HREF)
			return *u, err
		}
	}
	return url.URL{}, fmt.Errorf("couldn't find a Service List in current session")
}

func (c *VAClient) vaacquireservice(s url.URL, cid string) (u url.URL, err error) {

	if cid == "" {
		cid = os.Getenv("VCLOUDAIR_COMPUTEID")
	}

	req := c.Client.NewRequest(map[string]string{}, "GET", s, nil)

	// Add the Accept header for vCA
	req.Header.Add("Accept", "application/xml;version=5.6")

	// Set Authorization Header for vCA
	req.Header.Add("x-vchs-authorization", c.VAToken)

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return url.URL{}, fmt.Errorf("error processing compute action: %s", err)
	}

	services := new(services)

	if err = decodeBody(resp, services); err != nil {
		return url.URL{}, fmt.Errorf("error decoding services response: %s", err)
	}

	// Loop in the Services struct to find right service and compute resource.
	for _, s := range services.Service {
		if s.ServiceID == cid {
			c.Region = s.Region
			u, err := url.ParseRequestURI(s.HREF)
			return *u, err
		}
	}
	return url.URL{}, fmt.Errorf("couldn't find a Compute Resource in current service list")
}

func (c *VAClient) vaacquirecompute(s url.URL, vid string) (u url.URL, err error) {

	if vid == "" {
		vid = os.Getenv("VCLOUDAIR_VDCID")
	}

	req := c.Client.NewRequest(map[string]string{}, "GET", s, nil)

	// Add the Accept header for vCA
	req.Header.Add("Accept", "application/xml;version=5.6")

	// Set Authorization Header
	req.Header.Add("x-vchs-authorization", c.VAToken)

	// TODO: wrap into checkresp to parse error
	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return url.URL{}, fmt.Errorf("error processing compute action: %s", err)
	}

	computeresources := new(computeResources)

	if err = decodeBody(resp, computeresources); err != nil {
		return url.URL{}, fmt.Errorf("error decoding computeresources response: %s", err)
	}

	// Iterate through the ComputeResources struct searching for the right
	// backend server
	for _, s := range computeresources.VdcRef {
		if s.Name == vid {
			for _, t := range s.Link {
				if t.Name == vid {
					u, err := url.ParseRequestURI(t.HREF)
					return *u, err
				}
			}
		}
	}
	return url.URL{}, fmt.Errorf("couldn't find a VDC Resource in current Compute list")
}

func (c *VAClient) vagetbackendauth(s url.URL, cid string) error {

	if cid == "" {
		cid = os.Getenv("VCLOUDAIR_COMPUTEID")
	}

	req := c.Client.NewRequest(map[string]string{}, "POST", s, nil)

	// Add the Accept header for vCA
	req.Header.Add("Accept", "application/xml;version=5.6")

	// Set Authorization Header
	req.Header.Add("x-vchs-authorization", c.VAToken)

	// TODO: wrap into checkresp to parse error
	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return fmt.Errorf("error processing backend url action: %s", err)
	}
	defer resp.Body.Close()

	vcloudsession := new(vCloudSession)

	if err = decodeBody(resp, vcloudsession); err != nil {
		return fmt.Errorf("error decoding vcloudsession response: %s", err)
	}

	// Get the backend session information
	for _, s := range vcloudsession.VdcLink {
		if s.Name == cid {
			// Fetch the authorization token
			c.Client.VCDToken = s.AuthorizationToken

			// Fetch the authorization header
			c.Client.VCDAuthHeader = s.AuthorizationHeader

			u, err := url.ParseRequestURI(s.HREF)
			if err != nil {
				return fmt.Errorf("error decoding href: %s", err)
			}
			c.Client.VCDVDCHREF = *u
			return nil
		}
	}
	return fmt.Errorf("error finding the right backend resource")
}

// NewVAClient returns a new empty client to authenticate against the vCloud Air
// service, the vCloud Air endpoint can be overridden by setting the
// VCLOUDAIR_ENDPOINT environment variable.
func NewVAClient() (*VAClient, error) {

	var u *url.URL
	var err error

	if os.Getenv("VCLOUDAIR_ENDPOINT") != "" {
		u, err = url.ParseRequestURI(os.Getenv("VCLOUDAIR_ENDPOINT"))
		if err != nil {
			return &VAClient{}, fmt.Errorf("cannot parse endpoint coming from VCLOUDAIR_ENDPOINT")
		}
	} else {
		// Implicitly trust this URL parse.
		u, _ = url.ParseRequestURI("https://vchs.vmware.com/api")
	}

	VAClient := VAClient{
		VAEndpoint: *u,
		Client: Client{
			APIVersion: "5.6",
			// Patching things up as we're hitting several TLS timeouts.
			Http: http.Client{
				Transport: &http.Transport{
					Proxy:               http.ProxyFromEnvironment,
					TLSHandshakeTimeout: 120 * time.Second,
				},
			},
		},
	}
	return &VAClient, nil
}

// Authenticate is an helper function that performs a complete login in vCloud
// Air and in the backend vCloud Director instance.
func (c *VAClient) Authenticate(username, password, computeid, vdcid string) (Vdc, error) {
	// Authorize
	vaservicehref, err := c.vaauthorize(username, password)
	if err != nil {
		return Vdc{}, fmt.Errorf("error Authorizing: %s", err)
	}

	// Get Service
	vacomputehref, err := c.vaacquireservice(vaservicehref, computeid)
	if err != nil {
		return Vdc{}, fmt.Errorf("error Acquiring Service: %s", err)
	}

	// Get Compute
	vavdchref, err := c.vaacquirecompute(vacomputehref, vdcid)
	if err != nil {
		return Vdc{}, fmt.Errorf("error Acquiring Compute: %s", err)
	}

	// Get Backend Authorization
	if err = c.vagetbackendauth(vavdchref, computeid); err != nil {
		return Vdc{}, fmt.Errorf("error Acquiring Backend Authorization: %s", err)
	}

	v, err := c.Client.retrieveVDC()
	if err != nil {
		return Vdc{}, fmt.Errorf("error Acquiring VDC: %s", err)
	}

	return v, nil

}

// Disconnect performs a disconnection from the vCloud Air API endpoint.
func (c *VAClient) Disconnect() error {
	if c.Client.VCDToken == "" && c.Client.VCDAuthHeader == "" && c.VAToken == "" {
		return fmt.Errorf("cannot disconnect, client is not authenticated")
	}

	s := c.VAEndpoint
	s.Path += "/vchs/session"

	req := c.Client.NewRequest(map[string]string{}, "DELETE", s, nil)

	// Add the Accept header for vCA
	req.Header.Add("Accept", "application/xml;version=5.6")

	// Set Authorization Header
	req.Header.Add("x-vchs-authorization", c.VAToken)

	if _, err := checkResp(c.Client.Http.Do(req)); err != nil {
		return fmt.Errorf("error processing session delete for vchs: %s", err)
	}

	return nil
}
