package getproviders

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	svchost "github.com/hashicorp/terraform-svchost"
	svcauth "github.com/hashicorp/terraform-svchost/auth"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/httpclient"
	"github.com/hashicorp/terraform/version"
)

const (
	terraformVersionHeader = "X-Terraform-Version"

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
	discoveryRetry int
	requestTimeout time.Duration
)

func init() {
	configureDiscoveryRetry()
	configureRequestTimeout()
}

var SupportedPluginProtocols = MustParseVersionConstraints("~> 5")

// registryClient is a client for the provider registry protocol that is
// specialized only for the needs of this package. It's not intended as a
// general registry API client.
type registryClient struct {
	baseURL *url.URL
	creds   svcauth.HostCredentials

	httpClient *retryablehttp.Client
}

func newRegistryClient(baseURL *url.URL, creds svcauth.HostCredentials) *registryClient {
	httpClient := httpclient.New()
	httpClient.Timeout = requestTimeout

	retryableClient := retryablehttp.NewClient()
	retryableClient.HTTPClient = httpClient
	retryableClient.RetryMax = discoveryRetry
	retryableClient.RequestLogHook = requestLogHook
	retryableClient.ErrorHandler = maxRetryErrorHandler

	logOutput, err := logging.LogOutput()
	if err != nil {
		log.Printf("[WARN] Failed to set up registry client logger, "+
			"continuing without client logging: %s", err)
	}
	retryableClient.Logger = log.New(logOutput, "", log.Flags())

	return &registryClient{
		baseURL:    baseURL,
		creds:      creds,
		httpClient: retryableClient,
	}
}

// ProviderVersions returns the raw version and protocol strings produced by the
// registry for the given provider.
//
// The returned error will be ErrRegistryProviderNotKnown if the registry responds with
// 404 Not Found to indicate that the namespace or provider type are not known,
// ErrUnauthorized if the registry responds with 401 or 403 status codes, or
// ErrQueryFailed for any other protocol or operational problem.
func (c *registryClient) ProviderVersions(addr addrs.Provider) (map[string][]string, []string, error) {
	endpointPath, err := url.Parse(path.Join(addr.Namespace, addr.Type, "versions"))
	if err != nil {
		// Should never happen because we're constructing this from
		// already-validated components.
		return nil, nil, err
	}
	endpointURL := c.baseURL.ResolveReference(endpointPath)
	req, err := retryablehttp.NewRequest("GET", endpointURL.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	c.addHeadersToRequest(req.Request)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, c.errQueryFailed(addr, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// Great!
	case http.StatusNotFound:
		return nil, nil, ErrRegistryProviderNotKnown{
			Provider: addr,
		}
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, nil, c.errUnauthorized(addr.Hostname)
	default:
		return nil, nil, c.errQueryFailed(addr, errors.New(resp.Status))
	}

	// We ignore the platforms portion of the response body, because the
	// installer verifies the platform compatibility after pulling a provider
	// versions' metadata.
	type ResponseBody struct {
		Versions []struct {
			Version   string   `json:"version"`
			Protocols []string `json:"protocols"`
		} `json:"versions"`
		Warnings []string `json:"warnings"`
	}
	var body ResponseBody

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&body); err != nil {
		return nil, nil, c.errQueryFailed(addr, err)
	}

	if len(body.Versions) == 0 {
		return nil, body.Warnings, nil
	}

	ret := make(map[string][]string, len(body.Versions))
	for _, v := range body.Versions {
		ret[v.Version] = v.Protocols
	}

	return ret, body.Warnings, nil
}

// PackageMeta returns metadata about a distribution package for a provider.
//
// The returned error will be one of the following:
//
//   - ErrPlatformNotSupported if the registry responds with 404 Not Found,
//     under the assumption that the caller previously checked that the provider
//     and version are valid.
//   - ErrProtocolNotSupported if the requested provider version's protocols are not
//     supported by this version of terraform.
//   - ErrUnauthorized if the registry responds with 401 or 403 status codes
//   - ErrQueryFailed for any other operational problem.
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

	req, err := retryablehttp.NewRequest("GET", endpointURL.String(), nil)
	if err != nil {
		return PackageMeta{}, err
	}
	c.addHeadersToRequest(req.Request)

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

	type SigningKeyList struct {
		GPGPublicKeys []*SigningKey `json:"gpg_public_keys"`
	}
	type ResponseBody struct {
		Protocols   []string `json:"protocols"`
		OS          string   `json:"os"`
		Arch        string   `json:"arch"`
		Filename    string   `json:"filename"`
		DownloadURL string   `json:"download_url"`
		SHA256Sum   string   `json:"shasum"`

		SHA256SumsURL          string `json:"shasums_url"`
		SHA256SumsSignatureURL string `json:"shasums_signature_url"`

		SigningKeys SigningKeyList `json:"signing_keys"`
	}
	var body ResponseBody

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&body); err != nil {
		return PackageMeta{}, c.errQueryFailed(provider, err)
	}

	var protoVersions VersionList
	for _, versionStr := range body.Protocols {
		v, err := ParseVersion(versionStr)
		if err != nil {
			return PackageMeta{}, c.errQueryFailed(
				provider,
				fmt.Errorf("registry response includes invalid version string %q: %s", versionStr, err),
			)
		}
		protoVersions = append(protoVersions, v)
	}
	protoVersions.Sort()

	// Verify that this version of terraform supports the providers' protocol
	// version(s)
	if len(protoVersions) > 0 {
		supportedProtos := MeetingConstraints(SupportedPluginProtocols)
		protoErr := ErrProtocolNotSupported{
			Provider: provider,
			Version:  version,
		}
		match := false
		for _, version := range protoVersions {
			if supportedProtos.Has(version) {
				match = true
			}
		}
		if match == false {
			// If the protocol version is not supported, try to find the closest
			// matching version.
			closest, err := c.findClosestProtocolCompatibleVersion(provider, version)
			if err != nil {
				return PackageMeta{}, err
			}
			protoErr.Suggestion = closest
			return PackageMeta{}, protoErr
		}
	}

	downloadURL, err := url.Parse(body.DownloadURL)
	if err != nil {
		return PackageMeta{}, fmt.Errorf("registry response includes invalid download URL: %s", err)
	}
	downloadURL = resp.Request.URL.ResolveReference(downloadURL)
	if downloadURL.Scheme != "http" && downloadURL.Scheme != "https" {
		return PackageMeta{}, fmt.Errorf("registry response includes invalid download URL: must use http or https scheme")
	}

	ret := PackageMeta{
		Provider:         provider,
		Version:          version,
		ProtocolVersions: protoVersions,
		TargetPlatform: Platform{
			OS:   body.OS,
			Arch: body.Arch,
		},
		Filename: body.Filename,
		Location: PackageHTTPURL(downloadURL.String()),
		// "Authentication" is populated below
	}

	if len(body.SHA256Sum) != sha256.Size*2 { // *2 because it's hex-encoded
		return PackageMeta{}, c.errQueryFailed(
			provider,
			fmt.Errorf("registry response includes invalid SHA256 hash %q: %s", body.SHA256Sum, err),
		)
	}

	var checksum [sha256.Size]byte
	_, err = hex.Decode(checksum[:], []byte(body.SHA256Sum))
	if err != nil {
		return PackageMeta{}, c.errQueryFailed(
			provider,
			fmt.Errorf("registry response includes invalid SHA256 hash %q: %s", body.SHA256Sum, err),
		)
	}

	shasumsURL, err := url.Parse(body.SHA256SumsURL)
	if err != nil {
		return PackageMeta{}, fmt.Errorf("registry response includes invalid SHASUMS URL: %s", err)
	}
	shasumsURL = resp.Request.URL.ResolveReference(shasumsURL)
	if shasumsURL.Scheme != "http" && shasumsURL.Scheme != "https" {
		return PackageMeta{}, fmt.Errorf("registry response includes invalid SHASUMS URL: must use http or https scheme")
	}
	document, err := c.getFile(shasumsURL)
	if err != nil {
		return PackageMeta{}, c.errQueryFailed(
			provider,
			fmt.Errorf("failed to retrieve authentication checksums for provider: %s", err),
		)
	}
	signatureURL, err := url.Parse(body.SHA256SumsSignatureURL)
	if err != nil {
		return PackageMeta{}, fmt.Errorf("registry response includes invalid SHASUMS signature URL: %s", err)
	}
	signatureURL = resp.Request.URL.ResolveReference(signatureURL)
	if signatureURL.Scheme != "http" && signatureURL.Scheme != "https" {
		return PackageMeta{}, fmt.Errorf("registry response includes invalid SHASUMS signature URL: must use http or https scheme")
	}
	signature, err := c.getFile(signatureURL)
	if err != nil {
		return PackageMeta{}, c.errQueryFailed(
			provider,
			fmt.Errorf("failed to retrieve cryptographic signature for provider: %s", err),
		)
	}

	keys := make([]SigningKey, len(body.SigningKeys.GPGPublicKeys))
	for i, key := range body.SigningKeys.GPGPublicKeys {
		keys[i] = *key
	}

	ret.Authentication = PackageAuthenticationAll(
		NewMatchingChecksumAuthentication(document, body.Filename, checksum),
		NewArchiveChecksumAuthentication(checksum),
		NewSignatureAuthentication(document, signature, keys),
	)

	return ret, nil
}

// findClosestProtocolCompatibleVersion searches for the provider version with the closest protocol match.
func (c *registryClient) findClosestProtocolCompatibleVersion(provider addrs.Provider, version Version) (Version, error) {
	var match Version
	available, _, err := c.ProviderVersions(provider)
	if err != nil {
		return UnspecifiedVersion, err
	}

	// extract the maps keys so we can make a sorted list of available versions.
	versionList := make(VersionList, 0, len(available))
	for versionStr := range available {
		v, err := ParseVersion(versionStr)
		if err != nil {
			return UnspecifiedVersion, ErrQueryFailed{
				Provider: provider,
				Wrapped:  fmt.Errorf("registry response includes invalid version string %q: %s", versionStr, err),
			}
		}
		versionList = append(versionList, v)
	}
	versionList.Sort() // lowest precedence first, preserving order when equal precedence

	protoVersions := MeetingConstraints(SupportedPluginProtocols)
FindMatch:
	// put the versions in increasing order of precedence
	for index := len(versionList) - 1; index >= 0; index-- { // walk backwards to consider newer versions first
		for _, protoStr := range available[versionList[index].String()] {
			p, err := ParseVersion(protoStr)
			if err != nil {
				return UnspecifiedVersion, ErrQueryFailed{
					Provider: provider,
					Wrapped:  fmt.Errorf("registry response includes invalid protocol string %q: %s", protoStr, err),
				}
			}
			if protoVersions.Has(p) {
				match = versionList[index]
				break FindMatch
			}
		}
	}
	return match, nil
}

// LegacyProviderDefaultNamespace returns the raw address strings produced by
// the registry when asked about the given unqualified provider type name.
// The returned namespace string is taken verbatim from the registry's response.
//
// This method exists only to allow compatibility with unqualified names
// in older configurations. New configurations should be written so as not to
// depend on it.
func (c *registryClient) LegacyProviderDefaultNamespace(typeName string) (string, string, error) {
	endpointPath, err := url.Parse(path.Join("-", typeName, "versions"))
	if err != nil {
		// Should never happen because we're constructing this from
		// already-validated components.
		return "", "", err
	}
	endpointURL := c.baseURL.ResolveReference(endpointPath)

	req, err := retryablehttp.NewRequest("GET", endpointURL.String(), nil)
	if err != nil {
		return "", "", err
	}
	c.addHeadersToRequest(req.Request)

	// This is just to give us something to return in error messages. It's
	// not a proper provider address.
	placeholderProviderAddr := addrs.NewLegacyProvider(typeName)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", c.errQueryFailed(placeholderProviderAddr, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// Great!
	case http.StatusNotFound:
		return "", "", ErrProviderNotFound{
			Provider: placeholderProviderAddr,
		}
	case http.StatusUnauthorized, http.StatusForbidden:
		return "", "", c.errUnauthorized(placeholderProviderAddr.Hostname)
	default:
		return "", "", c.errQueryFailed(placeholderProviderAddr, errors.New(resp.Status))
	}

	type ResponseBody struct {
		Id      string `json:"id"`
		MovedTo string `json:"moved_to"`
	}
	var body ResponseBody

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&body); err != nil {
		return "", "", c.errQueryFailed(placeholderProviderAddr, err)
	}

	provider, diags := addrs.ParseProviderSourceString(body.Id)
	if diags.HasErrors() {
		return "", "", fmt.Errorf("Error parsing provider ID from Registry: %s", diags.Err())
	}

	if provider.Type != typeName {
		return "", "", fmt.Errorf("Registry returned provider with type %q, expected %q", provider.Type, typeName)
	}

	var movedTo addrs.Provider
	if body.MovedTo != "" {
		movedTo, diags = addrs.ParseProviderSourceString(body.MovedTo)
		if diags.HasErrors() {
			return "", "", fmt.Errorf("Error parsing provider ID from Registry: %s", diags.Err())
		}

		if movedTo.Type != typeName {
			return "", "", fmt.Errorf("Registry returned provider with type %q, expected %q", movedTo.Type, typeName)
		}
	}

	return provider.Namespace, movedTo.Namespace, nil
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

func (c *registryClient) getFile(url *url.URL) ([]byte, error) {
	resp, err := c.httpClient.Get(url.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return data, err
	}

	return data, nil
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

	// Additional error detail: if we have a response, use the status code;
	// if we have an error, use that; otherwise nothing. We will never have
	// both response and error.
	var errMsg string
	if resp != nil {
		errMsg = fmt.Sprintf(": %d", resp.StatusCode)
	} else if err != nil {
		errMsg = fmt.Sprintf(": %s", err)
	}

	// This function is always called with numTries=RetryMax+1. If we made any
	// retry attempts, include that in the error message.
	if numTries > 1 {
		return resp, fmt.Errorf("the request failed after %d attempts, please try again later%s",
			numTries, errMsg)
	}
	return resp, fmt.Errorf("the request failed, please try again later%s", errMsg)
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
