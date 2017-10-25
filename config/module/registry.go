package module

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	cleanhttp "github.com/hashicorp/go-cleanhttp"

	"github.com/hashicorp/terraform/registry/regsrc"
	"github.com/hashicorp/terraform/registry/response"
	"github.com/hashicorp/terraform/svchost"
	"github.com/hashicorp/terraform/svchost/disco"
	"github.com/hashicorp/terraform/version"
)

const (
	defaultRegistry   = "registry.terraform.io"
	defaultApiPath    = "/v1/modules"
	registryServiceID = "registry.v1"
	xTerraformGet     = "X-Terraform-Get"
	xTerraformVersion = "X-Terraform-Version"
	requestTimeout    = 10 * time.Second
	serviceID         = "modules.v1"
)

var (
	httpClient *http.Client
	tfVersion  = version.String()
	regDisco   = disco.NewDisco()
)

func init() {
	httpClient = cleanhttp.DefaultPooledClient()
	httpClient.Timeout = requestTimeout
}

type errModuleNotFound string

func (e errModuleNotFound) Error() string {
	return `module "` + string(e) + `" not found`
}

func discoverRegURL(d *disco.Disco, module *regsrc.Module) string {
	if d == nil {
		d = regDisco
	}

	if module.RawHost == nil {
		module.RawHost = regsrc.NewFriendlyHost(defaultRegistry)
	}

	regURL := d.DiscoverServiceURL(svchost.Hostname(module.RawHost.Normalized()), serviceID)
	if regURL == nil {
		regURL = &url.URL{
			Scheme: "https",
			Host:   module.RawHost.String(),
			Path:   defaultApiPath,
		}
	}

	service := regURL.String()

	if service[len(service)-1] != '/' {
		service += "/"
	}

	return service
}

// Lookup module versions in the registry.
func lookupModuleVersions(d *disco.Disco, module *regsrc.Module) (*response.ModuleVersions, error) {
	service := discoverRegURL(d, module)

	location := fmt.Sprintf("%s%s/versions", service, module.Module())
	log.Printf("[DEBUG] fetching module versions from %q", location)

	req, err := http.NewRequest("GET", location, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set(xTerraformVersion, tfVersion)

	if d == nil {
		d = regDisco
	}

	// if discovery required a custom transport, then we should use that too
	client := httpClient
	if d.Transport != nil {
		client = &http.Client{
			Transport: d.Transport,
			Timeout:   requestTimeout,
		}
	}

	resp, err := client.Do(req)
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
func lookupModuleLocation(d *disco.Disco, module *regsrc.Module, version string) (string, error) {
	service := discoverRegURL(d, module)

	var download string
	if version == "" {
		download = fmt.Sprintf("%s%s/download", service, module.Module())
	} else {
		download = fmt.Sprintf("%s%s/%s/download", service, module.Module(), version)
	}

	log.Printf("[DEBUG] looking up module location from %q", download)

	req, err := http.NewRequest("GET", download, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set(xTerraformVersion, tfVersion)

	// if discovery required a custom transport, then we should use that too
	client := httpClient
	if regDisco.Transport != nil {
		client = &http.Client{
			Transport: regDisco.Transport,
			Timeout:   requestTimeout,
		}
	}

	resp, err := client.Do(req)
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

	return location, nil
}
