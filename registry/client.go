package registry

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/httpclient"
	"github.com/hashicorp/terraform/registry/regsrc"
	"github.com/hashicorp/terraform/registry/response"
	"github.com/hashicorp/terraform/version"
)

const (
	xTerraformGet      = "X-Terraform-Get"
	xTerraformVersion  = "X-Terraform-Version"
	modulesServiceID   = "modules.v1"
	providersServiceID = "providers.v1"

	// registryDiscoveryRetryEnvName is the name of the environment variable that
	// can be configured to customize number of retries for module and provider
	// discovery requests with the remote registry.
	registryDiscoveryRetryEnvName = "TF_REGISTRY_DISCOVERY_RETRY"
	defaultRetry                  = 1

	// registryClientTimeoutEnvName is the name of the environment variable that
	// can be configured to customize the timeout duration (seconds) for module
	// and provider discovery with the remote registry.
	registryClientTimeoutEnvName = "TF_REGISTRY_CLIENT_TIMEOUT"

	// defaultRequestTimeout is the default timeout duration for requests to the
	// remote registry.
	defaultRequestTimeout = 10 * time.Second
)

var (
	tfVersion = version.String()

	discoveryRetry int
	requestTimeout time.Duration
)

func init() {
	configureDiscoveryRetry()
	configureRequestTimeout()
}

// Client provides methods to query Terraform Registries.
type Client struct {
	// this is the client to be used for all requests.
	client *retryablehttp.Client

	// services is a required *disco.Disco, which may have services and
	// credentials pre-loaded.
	services *disco.Disco

	// retry is the number of retries the client will attempt for each request
	// if it runs into a transient failure with the remote registry.
	retry int
}

// NewClient returns a new initialized registry client.
func NewClient(services *disco.Disco, client *http.Client) *Client {
	if services == nil {
		services = disco.New()
	}

	if client == nil {
		client = httpclient.New()
		client.Timeout = requestTimeout
	}
	retryableClient := retryablehttp.NewClient()
	retryableClient.HTTPClient = client
	retryableClient.RetryMax = discoveryRetry
	retryableClient.RequestLogHook = requestLogHook
	retryableClient.ErrorHandler = maxRetryErrorHandler

	logOutput, err := logging.LogOutput()
	if err != nil {
		log.Printf("[WARN] Failed to set up registry client logger, "+
			"continuing without client logging: %s", err)
	}
	retryableClient.Logger = log.New(logOutput, "", log.Flags())

	services.Transport = retryableClient.HTTPClient.Transport

	services.SetUserAgent(httpclient.TerraformUserAgent(version.String()))

	return &Client{
		client:   retryableClient,
		services: services,
	}
}

// Discover queries the host, and returns the url for the registry.
func (c *Client) Discover(host svchost.Hostname, serviceID string) (*url.URL, error) {
	service, err := c.services.DiscoverServiceURL(host, serviceID)
	if err != nil {
		return nil, &ServiceUnreachableError{err}
	}
	if !strings.HasSuffix(service.Path, "/") {
		service.Path += "/"
	}
	return service, nil
}

// ModuleVersions queries the registry for a module, and returns the available versions.
func (c *Client) ModuleVersions(module *regsrc.Module) (*response.ModuleVersions, error) {
	host, err := module.SvcHost()
	if err != nil {
		return nil, err
	}

	service, err := c.Discover(host, modulesServiceID)
	if err != nil {
		return nil, err
	}

	p, err := url.Parse(path.Join(module.Module(), "versions"))
	if err != nil {
		return nil, err
	}

	service = service.ResolveReference(p)

	log.Printf("[DEBUG] fetching module versions from %q", service)

	req, err := retryablehttp.NewRequest("GET", service.String(), nil)
	if err != nil {
		return nil, err
	}

	c.addRequestCreds(host, req.Request)
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
	creds, err := c.services.CredentialsForHost(host)
	if err != nil {
		log.Printf("[WARN] Failed to get credentials for %s: %s (ignoring)", host, err)
		return
	}

	if creds != nil {
		creds.PrepareRequest(req)
	}
}

// ModuleLocation find the download location for a specific version module.
// This returns a string, because the final location may contain special go-getter syntax.
func (c *Client) ModuleLocation(module *regsrc.Module, version string) (string, error) {
	host, err := module.SvcHost()
	if err != nil {
		return "", err
	}

	service, err := c.Discover(host, modulesServiceID)
	if err != nil {
		return "", err
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

	req, err := retryablehttp.NewRequest("GET", download.String(), nil)
	if err != nil {
		return "", err
	}

	c.addRequestCreds(host, req.Request)
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

// TerraformProviderVersions queries the registry for a provider, and returns the available versions.
func (c *Client) TerraformProviderVersions(provider *regsrc.TerraformProvider) (*response.TerraformProviderVersions, error) {
	host, err := provider.SvcHost()
	if err != nil {
		return nil, err
	}

	service, err := c.Discover(host, providersServiceID)
	if err != nil {
		return nil, err
	}

	p, err := url.Parse(path.Join(provider.TerraformProvider(), "versions"))
	if err != nil {
		return nil, err
	}

	service = service.ResolveReference(p)

	log.Printf("[DEBUG] fetching provider versions from %q", service)

	req, err := retryablehttp.NewRequest("GET", service.String(), nil)
	if err != nil {
		return nil, err
	}

	c.addRequestCreds(host, req.Request)
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
		return nil, &errProviderNotFound{addr: provider}
	default:
		return nil, fmt.Errorf("error looking up provider versions: %s", resp.Status)
	}

	var versions response.TerraformProviderVersions

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&versions); err != nil {
		return nil, err
	}

	return &versions, nil
}

// TerraformProviderLocation queries the registry for a provider download metadata
func (c *Client) TerraformProviderLocation(provider *regsrc.TerraformProvider, version string) (*response.TerraformProviderPlatformLocation, error) {
	host, err := provider.SvcHost()
	if err != nil {
		return nil, err
	}

	service, err := c.Discover(host, providersServiceID)
	if err != nil {
		return nil, err
	}

	p, err := url.Parse(path.Join(
		provider.TerraformProvider(),
		version,
		"download",
		provider.OS,
		provider.Arch,
	))
	if err != nil {
		return nil, err
	}

	service = service.ResolveReference(p)

	log.Printf("[DEBUG] fetching provider location from %q", service)

	req, err := retryablehttp.NewRequest("GET", service.String(), nil)
	if err != nil {
		return nil, err
	}

	c.addRequestCreds(host, req.Request)
	req.Header.Set(xTerraformVersion, tfVersion)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var loc response.TerraformProviderPlatformLocation

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&loc); err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		// OK
	case http.StatusNotFound:
		return nil, fmt.Errorf("provider %q version %q not found", provider.TerraformProvider(), version)
	default:
		// anything else is an error:
		return nil, fmt.Errorf("error getting download location for %q: %s", provider.TerraformProvider(), resp.Status)
	}

	return &loc, nil
}

// configureDiscoveryRetry configures the number of retries the registry client
// will attempt for requests with retryable errors, like 502 status codes
func configureDiscoveryRetry() {
	discoveryRetry = defaultRetry

	if v := os.Getenv(registryDiscoveryRetryEnvName); v != "" {
		retry, err := strconv.Atoi(v)
		if err == nil && retry > 0 {
			discoveryRetry = retry
		}
	}
}

func requestLogHook(logger retryablehttp.Logger, req *http.Request, i int) {
	if i > 0 {
		logger.Printf("[INFO] Previous request to the remote registry failed, attempting retry.")
	}
}

func maxRetryErrorHandler(resp *http.Response, err error, numTries int) (*http.Response, error) {
	// Close the body per library instructions
	if resp != nil {
		resp.Body.Close()
	}

	var errMsg string
	if err != nil {
		errMsg = fmt.Sprintf(" %s", err)
	}
	if numTries > 1 {
		return resp, fmt.Errorf("the request failed after %d attempts, please try again later: %d%s",
			numTries, resp.StatusCode, errMsg)
	}
	return resp, fmt.Errorf("the request failed, please try again later: %d%s", resp.StatusCode, errMsg)
}

// configureRequestTimeout configures the registry client request timeout from
// environment variables
func configureRequestTimeout() {
	requestTimeout = defaultRequestTimeout

	if v := os.Getenv(registryClientTimeoutEnvName); v != "" {
		timeout, err := strconv.Atoi(v)
		if err == nil && timeout > 0 {
			requestTimeout = time.Duration(timeout) * time.Second
		}
	}
}
