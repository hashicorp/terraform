package disco

import (
	"fmt"
	"net/url"
	"strings"
)

// Host represents a service discovered host.
type Host struct {
	discoURL *url.URL
	hostname string
	services map[string]interface{}
}

// ErrServiceNotProvided is returned when the service is not provided.
type ErrServiceNotProvided struct {
	hostname string
	service  string
}

// Error returns a customized error message.
func (e *ErrServiceNotProvided) Error() string {
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
	return fmt.Sprintf("host %s does not support %s version %s", e.hostname, e.service, e.version)
}

// ServiceURL returns the URL associated with the given service identifier,
// which should be of the form "servicename.vN".
//
// A non-nil result is always an absolute URL with a scheme of either HTTPS
// or HTTP.
func (h *Host) ServiceURL(id string) (*url.URL, error) {
	parts := strings.SplitN(id, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("Invalid service ID format (i.e. service.vN): %s", id)
	}
	service, version := parts[0], parts[1]

	// No services supported for an empty Host.
	if h == nil || h.services == nil {
		return nil, &ErrServiceNotProvided{hostname: "<unknown>", service: service}
	}

	urlStr, ok := h.services[id].(string)
	if !ok {
		// See if we have a matching service as that would indicate
		// the service is supported, but not the requested version.
		for serviceID := range h.services {
			if strings.HasPrefix(serviceID, service) {
				return nil, &ErrVersionNotSupported{
					hostname: h.hostname,
					service:  service,
					version:  version,
				}
			}
		}

		// No discovered services match the requested service ID.
		return nil, &ErrServiceNotProvided{hostname: h.hostname, service: service}
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
