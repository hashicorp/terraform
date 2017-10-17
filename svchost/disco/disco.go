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
	"github.com/hashicorp/terraform/terraform"
)

const (
	discoPath        = "/.well-known/terraform.json"
	maxRedirects     = 3               // arbitrary-but-small number to prevent runaway redirect loops
	discoTimeout     = 4 * time.Second // arbitrary-but-small time limit to prevent UI "hangs" during discovery
	maxDiscoDocBytes = 1 * 1024 * 1024 // 1MB - to prevent abusive services from using loads of our memory
)

var userAgent = fmt.Sprintf("Terraform/%s (service discovery)", terraform.VersionString())
var httpTransport = cleanhttp.DefaultPooledTransport() // overridden during tests, to skip TLS verification

// Disco is the main type in this package, which allows discovery on given
// hostnames and caches the results by hostname to avoid repeated requests
// for the same information.
type Disco struct {
	hostCache map[svchost.Hostname]Host
}

func NewDisco() *Disco {
	return &Disco{}
}

// Discover runs the discovery protocol against the given hostname (which must
// already have been validated and prepared with svchost.ForComparison) and
// returns an object describing the services available at that host.
//
// If a given hostname supports no Terraform services at all, a non-nil but
// empty Host object is returned. When giving feedback to the end user about
// such situations, we say e.g. "the host <name> doesn't provide a module
// registry", regardless of whether that is due to that service specifically
// being absent or due to the host not providing Terraform services at all,
// since we don't wish to expose the detail of whole-host discovery to an
// end-user.
func (d *Disco) Discover(host svchost.Hostname) Host {
	if d.hostCache == nil {
		d.hostCache = map[svchost.Hostname]Host{}
	}
	if cache, cached := d.hostCache[host]; cached {
		return cache
	}

	ret := d.discover(host)
	d.hostCache[host] = ret
	return ret
}

// DiscoverServiceURL is a convenience wrapper for discovery on a given
// hostname and then looking up a particular service in the result.
func (d *Disco) DiscoverServiceURL(host svchost.Hostname, serviceID string) *url.URL {
	return d.Discover(host).ServiceURL(serviceID)
}

// discover implements the actual discovery process, with its result cached
// by the public-facing Discover method.
func (d *Disco) discover(host svchost.Hostname) Host {
	discoURL := &url.URL{
		Scheme: "https",
		Host:   string(host),
		Path:   discoPath,
	}
	client := &http.Client{
		Transport: httpTransport,
		Timeout:   discoTimeout,

		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			log.Printf("[DEBUG] Service discovery redirected to %s", req.URL)
			if len(via) > maxRedirects {
				return errors.New("too many redirects") // (this error message will never actually be seen)
			}
			return nil
		},
	}

	var header = http.Header{}
	header.Set("User-Agent", userAgent)
	// TODO: look up credentials and add them to the header if we have them

	req := &http.Request{
		Method: "GET",
		URL:    discoURL,
		Header: header,
	}

	log.Printf("[DEBUG] Service discovery for %s at %s", host, discoURL)

	ret := Host{
		discoURL: discoURL,
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[WARNING] Failed to request discovery document: %s", err)
		return ret // empty
	}
	if resp.StatusCode != 200 {
		log.Printf("[WARNING] Failed to request discovery document: %s", resp.Status)
		return ret // empty
	}

	// If the client followed any redirects, we will have a new URL to use
	// as our base for relative resolution.
	ret.discoURL = resp.Request.URL

	contentType := resp.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		log.Printf("[WARNING] Discovery URL has malformed Content-Type %q", contentType)
		return ret // empty
	}
	if mediaType != "application/json" {
		log.Printf("[DEBUG] Discovery URL returned Content-Type %q, rather than application/json", mediaType)
		return ret // empty
	}

	// (this doesn't catch chunked encoding, because ContentLength is -1 in that case...)
	if resp.ContentLength > maxDiscoDocBytes {
		// Size limit here is not a contractual requirement and so we may
		// adjust it over time if we find a different limit is warranted.
		log.Printf("[WARNING] Discovery doc response is too large (got %d bytes; limit %d)", resp.ContentLength, maxDiscoDocBytes)
		return ret // empty
	}

	// If the response is using chunked encoding then we can't predict
	// its size, but we'll at least prevent reading the entire thing into
	// memory.
	lr := io.LimitReader(resp.Body, maxDiscoDocBytes)

	servicesBytes, err := ioutil.ReadAll(lr)
	if err != nil {
		log.Printf("[WARNING] Error reading discovery document body: %s", err)
		return ret // empty
	}

	var services map[string]interface{}
	err = json.Unmarshal(servicesBytes, &services)
	if err != nil {
		log.Printf("[WARNING] Failed to decode discovery document as a JSON object: %s", err)
		return ret // empty
	}

	ret.services = services
	return ret
}

// Forget invalidates any cached record of the given hostname. If the host
// has no cache entry then this is a no-op.
func (d *Disco) Forget(host svchost.Hostname) {
	delete(d.hostCache, host)
}

// ForgetAll is like Forget, but for all of the hostnames that have cache entries.
func (d *Disco) ForgetAll() {
	d.hostCache = nil
}
