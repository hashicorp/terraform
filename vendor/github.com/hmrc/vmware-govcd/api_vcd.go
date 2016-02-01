package govcd

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

type VCDClient struct {
	OrgHREF     url.URL // vCloud Director OrgRef
	Org         Org     // Org
	OrgVdc      Vdc     // Org vDC
	Client      Client  // Client for the underlying VCD instance
	sessionHREF url.URL // HREF for the session API
	Mutex       sync.Mutex
}

type supportedVersions struct {
	VersionInfo struct {
		Version  string `xml:"Version"`
		LoginUrl string `xml:"LoginUrl"`
	} `xml:"VersionInfo"`
}

func (c *VCDClient) vcdloginurl() error {

	s := c.Client.VCDVDCHREF
	s.Path += "/versions"

	// No point in checking for errors here
	req := c.Client.NewRequest(map[string]string{}, "GET", s, nil)

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	supportedVersions := new(supportedVersions)

	err = decodeBody(resp, supportedVersions)

	if err != nil {
		return fmt.Errorf("error decoding versions response: %s", err)
	}

	u, err := url.Parse(supportedVersions.VersionInfo.LoginUrl)
	if err != nil {
		return fmt.Errorf("couldn't find a LoginUrl in versions")
	}
	c.sessionHREF = *u
	return nil
}

func (c *VCDClient) vcdauthorize(user, pass, org string) error {

	if user == "" {
		user = os.Getenv("VCLOUD_USERNAME")
	}

	if pass == "" {
		pass = os.Getenv("VCLOUD_PASSWORD")
	}

	if org == "" {
		org = os.Getenv("VCLOUD_ORG")
	}

	// No point in checking for errors here
	req := c.Client.NewRequest(map[string]string{}, "POST", c.sessionHREF, nil)

	// Set Basic Authentication Header
	req.SetBasicAuth(user+"@"+org, pass)

	// Add the Accept header for vCA
	req.Header.Add("Accept", "application/*+xml;version=5.5")

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Store the authentication header
	c.Client.VCDToken = resp.Header.Get("x-vcloud-authorization")
	c.Client.VCDAuthHeader = "x-vcloud-authorization"

	session := new(session)
	err = decodeBody(resp, session)

	if err != nil {
		fmt.Errorf("error decoding session response: %s", err)
	}

	org_found := false
	// Loop in the session struct to find the organization.
	for _, s := range session.Link {
		if s.Type == "application/vnd.vmware.vcloud.org+xml" && s.Rel == "down" {
			u, err := url.Parse(s.HREF)
			if err != nil {
				return fmt.Errorf("couldn't find a Organization in current session, %v", err)
			}
			c.OrgHREF = *u
			org_found = true
		}
	}
	if !org_found {
		return fmt.Errorf("couldn't find a Organization in current session")
	}

	// Loop in the session struct to find the session url.
	session_found := false
	for _, s := range session.Link {
		if s.Rel == "remove" {
			u, err := url.Parse(s.HREF)
			if err != nil {
				return fmt.Errorf("couldn't find a logout HREF in current session, %v", err)
			}
			c.sessionHREF = *u
			session_found = true
		}
	}
	if !session_found {
		return fmt.Errorf("couldn't find a logout HREF in current session")
	}
	return nil
}

func (c *VCDClient) RetrieveOrg(vcdname string) (Org, error) {

	req := c.Client.NewRequest(map[string]string{}, "GET", c.OrgHREF, nil)
	req.Header.Add("Accept", "vnd.vmware.vcloud.org+xml;version=5.5")

	// TODO: wrap into checkresp to parse error
	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return Org{}, fmt.Errorf("error retreiving org: %s", err)
	}

	org := NewOrg(&c.Client)

	if err = decodeBody(resp, org.Org); err != nil {
		return Org{}, fmt.Errorf("error decoding org response: %s", err)
	}

	// Get the VDC ref from the Org
	for _, s := range org.Org.Link {
		if s.Type == "application/vnd.vmware.vcloud.vdc+xml" && s.Rel == "down" {
			if vcdname != "" && s.Name != vcdname {
				continue
			}
			u, err := url.Parse(s.HREF)
			if err != nil {
				return Org{}, err
			}
			c.Client.VCDVDCHREF = *u
		}
	}

	if &c.Client.VCDVDCHREF == nil {
		return Org{}, fmt.Errorf("error finding the organization VDC HREF")
	}

	return *org, nil
}

func NewVCDClient(vcdEndpoint url.URL, insecure bool) *VCDClient {

	return &VCDClient{
		Client: Client{
			APIVersion: "5.5",
			VCDVDCHREF: vcdEndpoint,
			Http: http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: insecure,
					},
					Proxy:               http.ProxyFromEnvironment,
					TLSHandshakeTimeout: 120 * time.Second,
				},
			},
		},
	}
}

// Authenticate is an helper function that performs a login in vCloud Director.
func (c *VCDClient) Authenticate(username, password, org, vdcname string) (Org, Vdc, error) {

	// LoginUrl
	err := c.vcdloginurl()
	if err != nil {
		return Org{}, Vdc{}, fmt.Errorf("error finding LoginUrl: %s", err)
	}
	// Authorize
	err = c.vcdauthorize(username, password, org)
	if err != nil {
		return Org{}, Vdc{}, fmt.Errorf("error authorizing: %s", err)
	}

	// Get Org
	o, err := c.RetrieveOrg(vdcname)
	if err != nil {
		return Org{}, Vdc{}, fmt.Errorf("error acquiring Org: %s", err)
	}

	vdc, err := c.Client.retrieveVDC()

	if err != nil {
		return Org{}, Vdc{}, fmt.Errorf("error retrieving the organization VDC")
	}

	return o, vdc, nil
}

// Disconnect performs a disconnection from the vCloud Director API endpoint.
func (c *VCDClient) Disconnect() error {
	if c.Client.VCDToken == "" && c.Client.VCDAuthHeader == "" {
		return fmt.Errorf("cannot disconnect, client is not authenticated")
	}

	req := c.Client.NewRequest(map[string]string{}, "DELETE", c.sessionHREF, nil)

	// Add the Accept header for vCA
	req.Header.Add("Accept", "application/xml;version=5.5")

	// Set Authorization Header
	req.Header.Add(c.Client.VCDAuthHeader, c.Client.VCDToken)

	if _, err := checkResp(c.Client.Http.Do(req)); err != nil {
		return fmt.Errorf("error processing session delete for vCloud Director: %s", err)
	}
	return nil
}
