package disco

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/httpclient"
)

const versionServiceID = "versions.v1"

// Host represents a service discovered host.
type Host struct {
	discoURL  *url.URL
	hostname  string
	services  map[string]interface{}
	transport http.RoundTripper
}

// Constraints represents the version constraints of a service.
type Constraints struct {
	Service   string   `json:"service"`
	Product   string   `json:"product"`
	Minimum   string   `json:"minimum"`
	Excluding []string `json:"excluding"`
	Maximum   string   `json:"maximum"`
}

// ErrServiceNotProvided is returned when the service is not provided.
type ErrServiceNotProvided struct {
	hostname string
	service  string
}

// Error returns a customized error message.
func (e *ErrServiceNotProvided) Error() string {
	if e.hostname == "" {
		return fmt.Sprintf("host does not provide a %s service", e.service)
	}
	return fmt.Sprintf("host %s does not provide a %s service", e.hostname, e.service)
}

// ErrVersionNotSupported is returned when the version is not supported.
type ErrVersionNotSupported struct {
	hostname string
	service  string
	version  string
}

// Error returns a customized error message.
func (e *ErrVersionNotSupported) Error() string {
	if e.hostname == "" {
		return fmt.Sprintf("host does not support %s version %s", e.service, e.version)
	}
	return fmt.Sprintf("host %s does not support %s version %s", e.hostname, e.service, e.version)
}

// ErrNoVersionConstraints is returned when checkpoint was disabled
// or the endpoint to query for version constraints was unavailable.
type ErrNoVersionConstraints struct {
	disabled bool
}

// Error returns a customized error message.
func (e *ErrNoVersionConstraints) Error() string {
	if e.disabled {
		return "checkpoint disabled"
	}
	return "unable to contact versions service"
}

// ServiceURL returns the URL associated with the given service identifier,
// which should be of the form "servicename.vN".
//
// A non-nil result is always an absolute URL with a scheme of either HTTPS
// or HTTP.
func (h *Host) ServiceURL(id string) (*url.URL, error) {
	svc, ver, err := parseServiceID(id)
	if err != nil {
		return nil, err
	}

	// No services supported for an empty Host.
	if h == nil || h.services == nil {
		return nil, &ErrServiceNotProvided{service: svc}
	}

	urlStr, ok := h.services[id].(string)
	if !ok {
		// See if we have a matching service as that would indicate
		// the service is supported, but not the requested version.
		for serviceID := range h.services {
			if strings.HasPrefix(serviceID, svc+".") {
				return nil, &ErrVersionNotSupported{
					hostname: h.hostname,
					service:  svc,
					version:  ver.Original(),
				}
			}
		}

		// No discovered services match the requested service.
		return nil, &ErrServiceNotProvided{hostname: h.hostname, service: svc}
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse service URL: %v", err)
	}

	// Make relative URLs absolute using our discovery URL.
	if !u.IsAbs() {
		u = h.discoURL.ResolveReference(u)
	}

	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, fmt.Errorf("Service URL is using an unsupported scheme: %s", u.Scheme)
	}
	if u.User != nil {
		return nil, fmt.Errorf("Embedded username/password information is not permitted")
	}

	// Fragment part is irrelevant, since we're not a browser.
	u.Fragment = ""

	return h.discoURL.ResolveReference(u), nil
}

// VersionConstraints returns the contraints for a given service identifier
// (which should be of the form "servicename.vN") and product.
//
// When an exact (service and version) match is found, the constraints for
// that service are returned.
//
// When the requested version is not provided but the service is, we will
// search for all alternative versions. If mutliple alternative versions
// are found, the contrains of the latest available version are returned.
//
// When a service is not provided at all an error will be returned instead.
//
// When checkpoint is disabled or when a 404 is returned after making the
// HTTP call, an ErrNoVersionConstraints error will be returned.
func (h *Host) VersionConstraints(id, product string) (*Constraints, error) {
	svc, _, err := parseServiceID(id)
	if err != nil {
		return nil, err
	}

	// Return early if checkpoint is disabled.
	if disabled := os.Getenv("CHECKPOINT_DISABLE"); disabled != "" {
		return nil, &ErrNoVersionConstraints{disabled: true}
	}

	// No services supported for an empty Host.
	if h == nil || h.services == nil {
		return nil, &ErrServiceNotProvided{service: svc}
	}

	// Try to get the service URL for the version service and
	// return early if the service isn't provided by the host.
	u, err := h.ServiceURL(versionServiceID)
	if err != nil {
		return nil, err
	}

	// Check if we have an exact (service and version) match.
	if _, ok := h.services[id].(string); !ok {
		// If we don't have an exact match, we search for all matching
		// services and then use the service ID of the latest version.
		var services []string
		for serviceID := range h.services {
			if strings.HasPrefix(serviceID, svc+".") {
				services = append(services, serviceID)
			}
		}

		if len(services) == 0 {
			// No discovered services match the requested service.
			return nil, &ErrServiceNotProvided{hostname: h.hostname, service: svc}
		}

		// Set id to the latest service ID we found.
		var latest *version.Version
		for _, serviceID := range services {
			if _, ver, err := parseServiceID(serviceID); err == nil {
				if latest == nil || latest.LessThan(ver) {
					id = serviceID
					latest = ver
				}
			}
		}
	}

	// Set a default timeout of 1 sec for the versions request (in milliseconds)
	timeout := 1000
	if _, err := strconv.Atoi(os.Getenv("CHECKPOINT_TIMEOUT")); err == nil {
		timeout, _ = strconv.Atoi(os.Getenv("CHECKPOINT_TIMEOUT"))
	}

	client := &http.Client{
		Transport: h.transport,
		Timeout:   time.Duration(timeout) * time.Millisecond,
	}

	// Prepare the service URL by setting the service and product.
	v := u.Query()
	v.Set("product", product)
	u.Path += id
	u.RawQuery = v.Encode()

	// Create a new request.
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create version constraints request: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", httpclient.UserAgentString())

	log.Printf("[DEBUG] Retrieve version constraints for service %s and product %s", id, product)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to request version constraints: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, &ErrNoVersionConstraints{disabled: false}
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Failed to request version constraints: %s", resp.Status)
	}

	// Parse the constraints from the response body.
	result := &Constraints{}
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return nil, fmt.Errorf("Error parsing version constraints: %v", err)
	}

	return result, nil
}

func parseServiceID(id string) (string, *version.Version, error) {
	parts := strings.SplitN(id, ".", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("Invalid service ID format (i.e. service.vN): %s", id)
	}

	version, err := version.NewVersion(parts[1])
	if err != nil {
		return "", nil, fmt.Errorf("Invalid service version: %v", err)
	}

	return parts[0], version, nil
}
