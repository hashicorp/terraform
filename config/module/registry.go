package module

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
	"github.com/hashicorp/terraform/version"
)

const (
	defaultRegistry   = "registry.terraform.io"
	registryServiceID = "registry.v1"
	xTerraformGet     = "X-Terraform-Get"
	xTerraformVersion = "X-Terraform-Version"
	requestTimeout    = 10 * time.Second
	serviceID         = "modules.v1"
)

var (
	httpClient *http.Client
	tfVersion  = version.String()
)

func init() {
	httpClient = cleanhttp.DefaultPooledClient()
	httpClient.Timeout = requestTimeout
}

type errModuleNotFound string

func (e errModuleNotFound) Error() string {
	return `module "` + string(e) + `" not found`
}

func (s *Storage) discoverRegURL(module *regsrc.Module) *url.URL {
	regURL := s.Services.DiscoverServiceURL(svchost.Hostname(module.RawHost.Normalized()), serviceID)
	if regURL == nil {
		return nil
	}

	if !strings.HasSuffix(regURL.Path, "/") {
		regURL.Path += "/"
	}

	return regURL
}

func (s *Storage) addRequestCreds(host svchost.Hostname, req *http.Request) {
	if s.Creds == nil {
		return
	}

	creds, err := s.Creds.ForHost(host)
	if err != nil {
		log.Printf("[WARNING] Failed to get credentials for %s: %s (ignoring)", host, err)
		return
	}

	if creds != nil {
		creds.PrepareRequest(req)
	}
}

// Lookup module versions in the registry.
func (s *Storage) lookupModuleVersions(module *regsrc.Module) (*response.ModuleVersions, error) {
	if module.RawHost == nil {
		module.RawHost = regsrc.NewFriendlyHost(defaultRegistry)
	}

	service := s.discoverRegURL(module)
	if service == nil {
		return nil, fmt.Errorf("host %s does not provide Terraform modules", module.RawHost.Display())
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

	s.addRequestCreds(svchost.Hostname(module.RawHost.Normalized()), req)
	req.Header.Set(xTerraformVersion, tfVersion)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// OK
	case http.StatusNotFound:
		return nil, errModuleNotFound(module.String())
	default:
		return nil, fmt.Errorf("error looking up module versions: %s", resp.Status)
	}

	var versions response.ModuleVersions

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&versions); err != nil {
		return nil, err
	}

	return &versions, nil
}

// lookup the location of a specific module version in the registry
func (s *Storage) lookupModuleLocation(module *regsrc.Module, version string) (string, error) {
	if module.RawHost == nil {
		module.RawHost = regsrc.NewFriendlyHost(defaultRegistry)
	}

	service := s.discoverRegURL(module)
	if service == nil {
		return "", fmt.Errorf("host %s does not provide Terraform modules", module.RawHost.Display())
	}

	var p *url.URL
	var err error
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

	s.addRequestCreds(svchost.Hostname(module.RawHost.Normalized()), req)
	req.Header.Set(xTerraformVersion, tfVersion)

	resp, err := httpClient.Do(req)
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
