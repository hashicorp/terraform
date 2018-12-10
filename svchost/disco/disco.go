// Package disco handles Terraform's remote service discovery protocol.
//
// This protocol allows mapping from a service hostname, as produced by the
// svchost package, to a set of services supported by that host and the
// endpoint information for each supported service.
package disco

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/url"
	"time"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform/svchost"
	"github.com/hashicorp/terraform/svchost/auth"
)

const (
	// Fixed path to the discovery manifest.
	discoPath = "/.well-known/terraform.json"

	// Arbitrary-but-small number to prevent runaway redirect loops.
	maxRedirects = 3

	// Arbitrary-but-small time limit to prevent UI "hangs" during discovery.
	discoTimeout = 11 * time.Second

	// 1MB - to prevent abusive services from using loads of our memory.
	maxDiscoDocBytes = 1 * 1024 * 1024
)

// httpTransport is overridden during tests, to skip TLS verification.
var httpTransport = cleanhttp.DefaultPooledTransport()

// Disco is the main type in this package, which allows discovery on given
// hostnames and caches the results by hostname to avoid repeated requests
// for the same information.
type Disco struct {
	hostCache map[svchost.Hostname]*Host
	credsSrc  auth.CredentialsSource

	// Transport is a custom http.RoundTripper to use.
	// A package default is used if this is nil.
	Transport http.RoundTripper
}

// New returns a new initialized discovery object.
func New() *Disco {
	return NewWithCredentialsSource(nil)
}

// NewWithCredentialsSource returns a new discovery object initialized with
// the given credentials source.
func NewWithCredentialsSource(credsSrc auth.CredentialsSource) *Disco {
	return &Disco{
		hostCache: make(map[svchost.Hostname]*Host),
		credsSrc:  credsSrc,
	}
}

// SetCredentialsSource provides a credentials source that will be used to
// add credentials to outgoing discovery requests, where available.
//
// If this method is never called, no outgoing discovery requests will have
// credentials.
func (d *Disco) SetCredentialsSource(src auth.CredentialsSource) {
	d.credsSrc = src
}

// CredentialsForHost returns a non-nil HostCredentials if the embedded source has
// credentials available for the host, and a nil HostCredentials if it does not.
func (d *Disco) CredentialsForHost(hostname svchost.Hostname) (auth.HostCredentials, error) {
	if d.credsSrc == nil {
		return nil, nil
	}
	return d.credsSrc.ForHost(hostname)
}

// ForceHostServices provides a pre-defined set of services for a given
// host, which prevents the receiver from attempting network-based discovery
// for the given host. Instead, the given services map will be returned
// verbatim.
//
// When providing "forced" services, any relative URLs are resolved against
// the initial discovery URL that would have been used for network-based
// discovery, yielding the same results as if the given map were published
// at the host's default discovery URL, though using absolute URLs is strongly
// recommended to make the configured behavior more explicit.
func (d *Disco) ForceHostServices(hostname svchost.Hostname, services map[string]interface{}) {
	if services == nil {
		services = map[string]interface{}{}
	}
	d.hostCache[hostname] = &Host{
		discoURL: &url.URL{
			Scheme: "https",
			Host:   string(hostname),
			Path:   discoPath,
		},
		hostname: hostname.ForDisplay(),
		services: services,
	}
}

// Discover runs the discovery protocol against the given hostname (which must
// already have been validated and prepared with svchost.ForComparison) and
// returns an object describing the services available at that host.
//
// If a given hostname supports no Terraform services at all, a non-nil but
// empty Host object is returned. When giving feedback to the end user about
// such situations, we say "host <name> does not provide a <service> service",
// regardless of whether that is due to that service specifically being absent
// or due to the host not providing Terraform services at all, since we don't
// wish to expose the detail of whole-host discovery to an end-user.
func (d *Disco) Discover(hostname svchost.Hostname) (*Host, error) {
	if host, cached := d.hostCache[hostname]; cached {
		return host, nil
	}

	host, err := d.discover(hostname)
	if err != nil {
		return nil, err
	}
	d.hostCache[hostname] = host

	return host, nil
}

// DiscoverServiceURL is a convenience wrapper for discovery on a given
// hostname and then looking up a particular service in the result.
func (d *Disco) DiscoverServiceURL(hostname svchost.Hostname, serviceID string) (*url.URL, error) {
	host, err := d.Discover(hostname)
	if err != nil {
		return nil, err
	}
	return host.ServiceURL(serviceID)
}

// discover implements the actual discovery process, with its result cached
// by the public-facing Discover method.
func (d *Disco) discover(hostname svchost.Hostname) (*Host, error) {
	discoURL := &url.URL{
		Scheme: "https",
		Host:   hostname.String(),
		Path:   discoPath,
	}

	t := d.Transport
	if t == nil {
		t = httpTransport
	}

	client := &http.Client{
		Transport: t,
		Timeout:   discoTimeout,

		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			log.Printf("[DEBUG] Service discovery redirected to %s", req.URL)
			if len(via) > maxRedirects {
				return errors.New("too many redirects") // this error will never actually be seen
			}
			return nil
		},
	}

	req := &http.Request{
		Method: "GET",
		URL:    discoURL,
	}

	creds, err := d.CredentialsForHost(hostname)
	if err != nil {
		log.Printf("[WARN] Failed to get credentials for %s: %s (ignoring)", hostname, err)
	}
	if creds != nil {
		// Update the request to include credentials.
		creds.PrepareRequest(req)
	}

	log.Printf("[DEBUG] Service discovery for %s at %s", hostname, discoURL)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to request discovery document: %v", err)
	}
	defer resp.Body.Close()

	host := &Host{
		// Use the discovery URL from resp.Request in
		// case the client followed any redirects.
		discoURL: resp.Request.URL,
		hostname: hostname.ForDisplay(),
	}

	// Return the host without any services.
	if resp.StatusCode == 404 {
		return host, nil
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Failed to request discovery document: %s", resp.Status)
	}

	contentType := resp.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("Discovery URL has a malformed Content-Type %q", contentType)
	}
	if mediaType != "application/json" {
		return nil, fmt.Errorf("Discovery URL returned an unsupported Content-Type %q", mediaType)
	}

	// This doesn't catch chunked encoding, because ContentLength is -1 in that case.
	if resp.ContentLength > maxDiscoDocBytes {
		// Size limit here is not a contractual requirement and so we may
		// adjust it over time if we find a different limit is warranted.
		return nil, fmt.Errorf(
			"Discovery doc response is too large (got %d bytes; limit %d)",
			resp.ContentLength, maxDiscoDocBytes,
		)
	}

	// If the response is using chunked encoding then we can't predict its
	// size, but we'll at least prevent reading the entire thing into memory.
	lr := io.LimitReader(resp.Body, maxDiscoDocBytes)

	servicesBytes, err := ioutil.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("Error reading discovery document body: %v", err)
	}

	var services map[string]interface{}
	err = json.Unmarshal(servicesBytes, &services)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode discovery document as a JSON object: %v", err)
	}
	host.services = services

	return host, nil
}

// Forget invalidates any cached record of the given hostname. If the host
// has no cache entry then this is a no-op.
func (d *Disco) Forget(hostname svchost.Hostname) {
	delete(d.hostCache, hostname)
}

// ForgetAll is like Forget, but for all of the hostnames that have cache entries.
func (d *Disco) ForgetAll() {
	d.hostCache = make(map[svchost.Hostname]*Host)
}
