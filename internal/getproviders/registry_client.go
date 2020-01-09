package getproviders

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/apparentlymart/go-versions/versions"
	svchost "github.com/hashicorp/terraform-svchost"
	svcauth "github.com/hashicorp/terraform-svchost/auth"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/httpclient"
	"github.com/hashicorp/terraform/version"
)

const terraformVersionHeader = "X-Terraform-Version"

// registryClient is a client for the provider registry protocol that is
// specialized only for the needs of this package. It's not intended as a
// general registry API client.
type registryClient struct {
	baseURL *url.URL
	creds   svcauth.HostCredentials

	httpClient *http.Client
}

func newRegistryClient(baseURL *url.URL, creds svcauth.HostCredentials) *registryClient {
	httpClient := httpclient.New()
	httpClient.Timeout = 10 * time.Second

	return &registryClient{
		baseURL:    baseURL,
		creds:      creds,
		httpClient: httpClient,
	}
}

// ProviderVersions returns the raw version strings produced by the registry
// for the given provider.
//
// The returned error will be ErrProviderNotKnown if the registry responds
// with 404 Not Found to indicate that the namespace or provider type are
// not known, ErrUnauthorized if the registry responds with 401 or 403 status
// codes, or ErrQueryFailed for any other protocol or operational problem.
func (c *registryClient) ProviderVersions(addr addrs.Provider) ([]string, error) {
	endpointPath, err := url.Parse(path.Join(addr.Namespace, addr.Type, "versions"))
	if err != nil {
		// Should never happen because we're constructing this from
		// already-validated components.
		return nil, err
	}
	endpointURL := c.baseURL.ResolveReference(endpointPath)

	req, err := http.NewRequest("GET", endpointURL.String(), nil)
	if err != nil {
		return nil, err
	}
	c.addHeadersToRequest(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.errQueryFailed(addr, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// Great!
	case http.StatusNotFound:
		return nil, ErrProviderNotKnown{
			Provider: addr,
		}
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, c.errUnauthorized(addr.Hostname)
	default:
		return nil, c.errQueryFailed(addr, errors.New(resp.Status))
	}

	// We ignore everything except the version numbers here because our goal
	// is to find out which versions are available _at all_. Which ones are
	// compatible with the current Terraform becomes relevant only once we've
	// selected one, at which point we'll return an error if the selected one
	// is incompatible.
	//
	// We intentionally produce an error on incompatibility, rather than
	// silently ignoring an incompatible version, in order to give the user
	// explicit feedback about why their selection wasn't valid and allow them
	// to decide whether to fix that by changing the selection or by some other
	// action such as upgrading Terraform, using a different OS to run
	// Terraform, etc. Changes that affect compatibility are considered
	// breaking changes from a provider API standpoint, so provider teams
	// should change compatibility only in new major versions.
	type ResponseBody struct {
		Versions []struct {
			Version string `json:"version"`
		} `json:"versions"`
	}
	var body ResponseBody

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&body); err != nil {
		return nil, c.errQueryFailed(addr, err)
	}

	if len(body.Versions) == 0 {
		return nil, nil
	}

	ret := make([]string, len(body.Versions))
	for i, v := range body.Versions {
		ret[i] = v.Version
	}
	return ret, nil
}

// PackageMeta returns metadata about a distribution package for a
// provider.
//
// The returned error will be ErrPlatformNotSupported if the registry responds
// with 404 Not Found, under the assumption that the caller previously checked
// that the provider and version are valid. It will return ErrUnauthorized if
// the registry responds with 401 or 403 status codes, or ErrQueryFailed for
// any other protocol or operational problem.
func (c *registryClient) PackageMeta(provider addrs.Provider, version Version, target Platform) (PackageMeta, error) {
	endpointPath, err := url.Parse(path.Join(
		provider.Namespace,
		provider.Type,
		version.String(),
		"download",
		target.OS,
		target.Arch,
	))
	if err != nil {
		// Should never happen because we're constructing this from
		// already-validated components.
		return PackageMeta{}, err
	}
	endpointURL := c.baseURL.ResolveReference(endpointPath)

	req, err := http.NewRequest("GET", endpointURL.String(), nil)
	if err != nil {
		return PackageMeta{}, err
	}
	c.addHeadersToRequest(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return PackageMeta{}, c.errQueryFailed(provider, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// Great!
	case http.StatusNotFound:
		return PackageMeta{}, ErrPlatformNotSupported{
			Provider: provider,
			Version:  version,
			Platform: target,
		}
	case http.StatusUnauthorized, http.StatusForbidden:
		return PackageMeta{}, c.errUnauthorized(provider.Hostname)
	default:
		return PackageMeta{}, c.errQueryFailed(provider, errors.New(resp.Status))
	}

	type ResponseBody struct {
		Protocols   []string `json:"protocols"`
		OS          string   `json:"os"`
		Arch        string   `json:"arch"`
		Filename    string   `json:"filename"`
		DownloadURL string   `json:"download_url"`
		SHA256Sum   string   `json:"shasum"`

		// TODO: Other metadata for signature checking
	}
	var body ResponseBody

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&body); err != nil {
		return PackageMeta{}, c.errQueryFailed(provider, err)
	}

	var protoVersions VersionList
	for _, versionStr := range body.Protocols {
		v, err := versions.ParseVersion(versionStr)
		if err != nil {
			return PackageMeta{}, c.errQueryFailed(
				provider,
				fmt.Errorf("registry response includes invalid version string %q: %s", versionStr, err),
			)
		}
		protoVersions = append(protoVersions, v)
	}
	protoVersions.Sort()

	ret := PackageMeta{
		ProtocolVersions: protoVersions,
		TargetPlatform: Platform{
			OS:   body.OS,
			Arch: body.Arch,
		},
		Filename: body.Filename,
		Location: PackageHTTPURL(body.DownloadURL),
		// SHA256Sum is populated below
	}

	if len(body.SHA256Sum) != len(ret.SHA256Sum)*2 {
		return PackageMeta{}, c.errQueryFailed(
			provider,
			fmt.Errorf("registry response includes invalid SHA256 hash %q: %s", body.SHA256Sum, err),
		)
	}
	_, err = hex.Decode(ret.SHA256Sum[:], []byte(body.SHA256Sum))
	if err != nil {
		return PackageMeta{}, c.errQueryFailed(
			provider,
			fmt.Errorf("registry response includes invalid SHA256 hash %q: %s", body.SHA256Sum, err),
		)
	}

	return ret, nil
}

func (c *registryClient) addHeadersToRequest(req *http.Request) {
	if c.creds != nil {
		c.creds.PrepareRequest(req)
	}
	req.Header.Set(terraformVersionHeader, version.String())
}

func (c *registryClient) errQueryFailed(provider addrs.Provider, err error) error {
	return ErrQueryFailed{
		Provider: provider,
		Wrapped:  err,
	}
}

func (c *registryClient) errUnauthorized(hostname svchost.Hostname) error {
	return ErrUnauthorized{
		Hostname:        hostname,
		HaveCredentials: c.creds != nil,
	}
}
