package registry

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform/registry/regsrc"
	"github.com/hashicorp/terraform/registry/response"
	"github.com/hashicorp/terraform/svchost"
	"github.com/hashicorp/terraform/svchost/auth"
	"github.com/hashicorp/terraform/svchost/disco"
	"github.com/hashicorp/terraform/version"
)

const (
	xTerraformGet     = "X-Terraform-Get"
	xTerraformVersion = "X-Terraform-Version"
	requestTimeout    = 10 * time.Second
	serviceID         = "modules.v1"
)

var tfVersion = version.String()

// Client provides methods to query Terraform Registries.
type Client struct {
	// this is the client to be used for all requests.
	client *http.Client

	// services is a required *disco.Disco, which may have services and
	// credentials pre-loaded.
	services *disco.Disco

	// Creds optionally provides credentials for communicating with service
	// providers.
	creds auth.CredentialsSource
}

func NewClient(services *disco.Disco, creds auth.CredentialsSource, client *http.Client) *Client {
	if services == nil {
		services = disco.NewDisco()
	}

	services.SetCredentialsSource(creds)

	if client == nil {
		client = cleanhttp.DefaultPooledClient()
		client.Timeout = requestTimeout
	}

	services.Transport = client.Transport

	return &Client{
		client:   client,
		services: services,
		creds:    creds,
	}
}

// Discover qeuries the host, and returns the url for the registry.
func (c *Client) Discover(host svchost.Hostname) *url.URL {
	service := c.services.DiscoverServiceURL(host, serviceID)
	if service == nil {
		return nil
	}
	if !strings.HasSuffix(service.Path, "/") {
		service.Path += "/"
	}
	return service
}

// Versions queries the registry for a module, and returns the available versions.
func (c *Client) Versions(module *regsrc.Module) (*response.ModuleVersions, error) {
	host, err := module.SvcHost()
	if err != nil {
		return nil, err
	}

	service := c.Discover(host)
	if service == nil {
		return nil, fmt.Errorf("host %s does not provide Terraform modules", host)
	}

	p, err := url.Parse(path.Join(module.Module(), "versions"))
	if err != nil {
		return nil, err
	}

	service = service.ResolveReference(p)

	log.Printf("[DEBUG] fetching module versions from %q", service)

	req, err := http.NewRequest("GET", service.String(), nil)
	if err != nil {
		return nil, err
	}

	c.addRequestCreds(host, req)
	req.Header.Set(xTerraformVersion, tfVersion)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// OK
	case http.StatusNotFound:
		return nil, &errModuleNotFound{addr: module}
	default:
		return nil, fmt.Errorf("error looking up module versions: %s", resp.Status)
	}

	var versions response.ModuleVersions

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&versions); err != nil {
		return nil, err
	}

	for _, mod := range versions.Modules {
		for _, v := range mod.Versions {
			log.Printf("[DEBUG] found available version %q for %s", v.Version, mod.Source)
		}
	}

	return &versions, nil
}

func (c *Client) addRequestCreds(host svchost.Hostname, req *http.Request) {
	if c.creds == nil {
		return
	}

	creds, err := c.creds.ForHost(host)
	if err != nil {
		log.Printf("[WARN] Failed to get credentials for %s: %s (ignoring)", host, err)
		return
	}

	if creds != nil {
		creds.PrepareRequest(req)
	}
}

// Location find the download location for a specific version module.
// This returns a string, because the final location may contain special go-getter syntax.
func (c *Client) Location(module *regsrc.Module, version string) (string, error) {
	host, err := module.SvcHost()
	if err != nil {
		return "", err
	}

	service := c.Discover(host)
	if service == nil {
		return "", fmt.Errorf("host %s does not provide Terraform modules", host.ForDisplay())
	}

	var p *url.URL
	if version == "" {
		p, err = url.Parse(path.Join(module.Module(), "download"))
	} else {
		p, err = url.Parse(path.Join(module.Module(), version, "download"))
	}
	if err != nil {
		return "", err
	}
	download := service.ResolveReference(p)

	log.Printf("[DEBUG] looking up module location from %q", download)

	req, err := http.NewRequest("GET", download.String(), nil)
	if err != nil {
		return "", err
	}

	c.addRequestCreds(host, req)
	req.Header.Set(xTerraformVersion, tfVersion)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// there should be no body, but save it for logging
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body from registry: %s", err)
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		// OK
	case http.StatusNotFound:
		return "", fmt.Errorf("module %q version %q not found", module, version)
	default:
		// anything else is an error:
		return "", fmt.Errorf("error getting download location for %q: %s resp:%s", module, resp.Status, body)
	}

	// the download location is in the X-Terraform-Get header
	location := resp.Header.Get(xTerraformGet)
	if location == "" {
		return "", fmt.Errorf("failed to get download URL for %q: %s resp:%s", module, resp.Status, body)
	}

	// If location looks like it's trying to be a relative URL, treat it as
	// one.
	//
	// We don't do this for just _any_ location, since the X-Terraform-Get
	// header is a go-getter location rather than a URL, and so not all
	// possible values will parse reasonably as URLs.)
	//
	// When used in conjunction with go-getter we normally require this header
	// to be an absolute URL, but we are more liberal here because third-party
	// registry implementations may not "know" their own absolute URLs if
	// e.g. they are running behind a reverse proxy frontend, or such.
	if strings.HasPrefix(location, "/") || strings.HasPrefix(location, "./") || strings.HasPrefix(location, "../") {
		locationURL, err := url.Parse(location)
		if err != nil {
			return "", fmt.Errorf("invalid relative URL for %q: %s", module, err)
		}
		locationURL = download.ResolveReference(locationURL)
		location = locationURL.String()
	}

	return location, nil
}
