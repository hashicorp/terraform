package module

import (
	"encoding/json"
	"fmt"
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
	client    *http.Client
	tfVersion = version.String()
)

func init() {
	client = cleanhttp.DefaultPooledClient()
	client.Timeout = requestTimeout
}

type errModuleNotFound string

func (e errModuleNotFound) Error() string {
	return `module "` + string(e) + `" not found`
}

// Lookup module versions in the registry.
func lookupModuleVersions(regDisco *disco.Disco, module *regsrc.Module) (*response.ModuleVersions, error) {
	if module.RawHost == nil {
		module.RawHost = regsrc.NewFriendlyHost(defaultRegistry)
	}

	regURL := regDisco.DiscoverServiceURL(svchost.Hostname(module.RawHost.Normalized()), serviceID)
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

	location := fmt.Sprintf("%s%s/%s/%s/versions", service, module.RawNamespace, module.RawName, module.RawProvider)
	log.Printf("[DEBUG] fetching module versions from %q", location)

	req, err := http.NewRequest("GET", location, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set(xTerraformVersion, tfVersion)

	// if discovery required a custom transport, then we should use that too
	client := client
	if regDisco.Transport != nil {
		client = &http.Client{
			Transport: regDisco.Transport,
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
