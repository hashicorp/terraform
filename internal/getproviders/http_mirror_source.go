package getproviders

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	svchost "github.com/hashicorp/terraform-svchost"
	svcauth "github.com/hashicorp/terraform-svchost/auth"
	"golang.org/x/net/idna"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/httpclient"
	"github.com/hashicorp/terraform/version"
)

// HTTPMirrorSource is a source that reads provider metadata from a provider
// mirror that is accessible over the HTTP provider mirror protocol.
type HTTPMirrorSource struct {
	baseURL    *url.URL
	creds      svcauth.CredentialsSource
	httpClient *retryablehttp.Client
}

var _ Source = (*HTTPMirrorSource)(nil)

// NewHTTPMirrorSource constructs and returns a new network mirror source with
// the given base URL. The relative URL offsets defined by the HTTP mirror
// protocol will be resolve relative to the given URL.
//
// The given URL must use the "https" scheme, or this function will panic.
// (When the URL comes from user input, such as in the CLI config, it's the
// UI/config layer's responsibility to validate this and return a suitable
// error message for the end-user audience.)
func NewHTTPMirrorSource(baseURL *url.URL, creds svcauth.CredentialsSource) *HTTPMirrorSource {
	httpClient := httpclient.New()
	httpClient.Timeout = requestTimeout
	httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		// If we get redirected more than five times we'll assume we're
		// in a redirect loop and bail out, rather than hanging forever.
		if len(via) > 5 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	}
	return newHTTPMirrorSourceWithHTTPClient(baseURL, creds, httpClient)
}

func newHTTPMirrorSourceWithHTTPClient(baseURL *url.URL, creds svcauth.CredentialsSource, httpClient *http.Client) *HTTPMirrorSource {
	if baseURL.Scheme != "https" {
		panic("non-https URL for HTTP mirror")
	}

	// We borrow the retry settings and behaviors from the registry client,
	// because our needs here are very similar to those of the registry client.
	retryableClient := retryablehttp.NewClient()
	retryableClient.HTTPClient = httpClient
	retryableClient.RetryMax = discoveryRetry
	retryableClient.RequestLogHook = requestLogHook
	retryableClient.ErrorHandler = maxRetryErrorHandler

	logOutput, err := logging.LogOutput()
	if err != nil {
		log.Printf("[WARN] Failed to set up provider HTTP mirror logger, so continuing without client logging: %s", err)
	}
	retryableClient.Logger = log.New(logOutput, "", log.Flags())

	return &HTTPMirrorSource{
		baseURL:    baseURL,
		creds:      creds,
		httpClient: retryableClient,
	}
}

// AvailableVersions retrieves the available versions for the given provider
// from the object's underlying HTTP mirror service.
func (s *HTTPMirrorSource) AvailableVersions(provider addrs.Provider) (VersionList, Warnings, error) {
	log.Printf("[DEBUG] Querying available versions of provider %s at network mirror %s", provider.String(), s.baseURL.String())

	endpointPath := path.Join(
		provider.Hostname.String(),
		provider.Namespace,
		provider.Type,
		"index.json",
	)

	statusCode, body, finalURL, err := s.get(endpointPath)
	defer func() {
		if body != nil {
			body.Close()
		}
	}()
	if err != nil {
		return nil, nil, s.errQueryFailed(provider, err)
	}

	switch statusCode {
	case http.StatusOK:
		// Great!
	case http.StatusNotFound:
		return nil, nil, ErrProviderNotFound{
			Provider: provider,
		}
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, nil, s.errUnauthorized(finalURL)
	default:
		return nil, nil, s.errQueryFailed(provider, fmt.Errorf("server returned unsuccessful status %d", statusCode))
	}

	// If we got here then the response had status OK and so our body
	// will be non-nil and should contain some JSON for us to parse.
	type ResponseBody struct {
		Versions map[string]struct{} `json:"versions"`
	}
	var bodyContent ResponseBody

	dec := json.NewDecoder(body)
	if err := dec.Decode(&bodyContent); err != nil {
		return nil, nil, s.errQueryFailed(provider, fmt.Errorf("invalid response content from mirror server: %s", err))
	}

	if len(bodyContent.Versions) == 0 {
		return nil, nil, nil
	}
	ret := make(VersionList, 0, len(bodyContent.Versions))
	for versionStr := range bodyContent.Versions {
		version, err := ParseVersion(versionStr)
		if err != nil {
			log.Printf("[WARN] Ignoring invalid %s version string %q in provider mirror response", provider, versionStr)
			continue
		}
		ret = append(ret, version)
	}

	ret.Sort()
	return ret, nil, nil
}

// PackageMeta retrieves metadata for the requested provider package
// from the object's underlying HTTP mirror service.
func (s *HTTPMirrorSource) PackageMeta(provider addrs.Provider, version Version, target Platform) (PackageMeta, error) {
	log.Printf("[DEBUG] Finding package URL for %s v%s on %s via network mirror %s", provider.String(), version.String(), target.String(), s.baseURL.String())

	endpointPath := path.Join(
		provider.Hostname.String(),
		provider.Namespace,
		provider.Type,
		version.String()+".json",
	)

	statusCode, body, finalURL, err := s.get(endpointPath)
	defer func() {
		if body != nil {
			body.Close()
		}
	}()
	if err != nil {
		return PackageMeta{}, s.errQueryFailed(provider, err)
	}

	switch statusCode {
	case http.StatusOK:
		// Great!
	case http.StatusNotFound:
		// A 404 Not Found for a version we previously saw in index.json is
		// a protocol error, so we'll report this as "query failed.
		return PackageMeta{}, s.errQueryFailed(provider, fmt.Errorf("provider mirror does not have archive index for previously-reported %s version %s", provider, version))
	case http.StatusUnauthorized, http.StatusForbidden:
		return PackageMeta{}, s.errUnauthorized(finalURL)
	default:
		return PackageMeta{}, s.errQueryFailed(provider, fmt.Errorf("server returned unsuccessful status %d", statusCode))
	}

	// If we got here then the response had status OK and so our body
	// will be non-nil and should contain some JSON for us to parse.
	type ResponseArchiveMeta struct {
		RelativeURL string `json:"url"`
		Hashes      []string
	}
	type ResponseBody struct {
		Archives map[string]*ResponseArchiveMeta `json:"archives"`
	}
	var bodyContent ResponseBody

	dec := json.NewDecoder(body)
	if err := dec.Decode(&bodyContent); err != nil {
		return PackageMeta{}, s.errQueryFailed(provider, fmt.Errorf("invalid response content from mirror server: %s", err))
	}

	archiveMeta, ok := bodyContent.Archives[target.String()]
	if !ok {
		return PackageMeta{}, ErrPlatformNotSupported{
			Provider:  provider,
			Version:   version,
			Platform:  target,
			MirrorURL: s.baseURL,
		}
	}

	relURL, err := url.Parse(archiveMeta.RelativeURL)
	if err != nil {
		return PackageMeta{}, s.errQueryFailed(
			provider,
			fmt.Errorf("provider mirror returned invalid URL %q: %s", archiveMeta.RelativeURL, err),
		)
	}
	absURL := finalURL.ResolveReference(relURL)

	ret := PackageMeta{
		Provider:       provider,
		Version:        version,
		TargetPlatform: target,

		Location: PackageHTTPURL(absURL.String()),
		Filename: path.Base(absURL.Path),
	}
	// A network mirror might not provide any hashes at all, in which case
	// the package has no source-defined authentication whatsoever.
	if len(archiveMeta.Hashes) > 0 {
		ret.Authentication = NewPackageHashAuthentication(archiveMeta.Hashes)
	}

	return ret, nil
}

// ForDisplay returns a string description of the source for user-facing output.
func (s *HTTPMirrorSource) ForDisplay(provider addrs.Provider) string {
	return "provider mirror at " + s.baseURL.String()
}

// mirrorHost extracts the hostname portion of the configured base URL and
// returns it as a svchost.Hostname, normalized in the usual ways.
//
// If the returned error is non-nil then the given hostname doesn't comply
// with the IETF RFC 5891 section 5.3 and 5.4 validation rules, and thus cannot
// be interpreted as a valid Terraform service host. The IDNA validation errors
// are unfortunately usually not very user-friendly, but they are also
// relatively rare because the IDNA normalization rules are quite tolerant.
func (s *HTTPMirrorSource) mirrorHost() (svchost.Hostname, error) {
	return svchostFromURL(s.baseURL)
}

// mirrorHostCredentials returns the HostCredentials, if any, for the hostname
// included in the mirror base URL.
//
// It might return an error if the mirror base URL is invalid, or if the
// credentials lookup itself fails.
func (s *HTTPMirrorSource) mirrorHostCredentials() (svcauth.HostCredentials, error) {
	hostname, err := s.mirrorHost()
	if err != nil {
		return nil, fmt.Errorf("invalid provider mirror base URL %s: %s", s.baseURL.String(), err)
	}

	if s.creds == nil {
		// No host-specific credentials, then.
		return nil, nil
	}

	return s.creds.ForHost(hostname)
}

// get is the shared functionality for querying a JSON index from a mirror.
//
// It only handles the raw HTTP request. The "body" return value is the
// reader from the response if and only if the response status code is 200 OK
// and the Content-Type is application/json. In all other cases it's nil.
// If body is non-nil then the caller must close it after reading it.
//
// If the "finalURL" return value is not empty then it's the URL that actually
// produced the returned response, possibly after following some redirects.
func (s *HTTPMirrorSource) get(relativePath string) (statusCode int, body io.ReadCloser, finalURL *url.URL, error error) {
	endpointPath, err := url.Parse(relativePath)
	if err != nil {
		// Should never happen because the caller should validate all of the
		// components it's including in the path.
		return 0, nil, nil, err
	}
	endpointURL := s.baseURL.ResolveReference(endpointPath)

	req, err := retryablehttp.NewRequest("GET", endpointURL.String(), nil)
	if err != nil {
		return 0, nil, endpointURL, err
	}
	req.Request.Header.Set(terraformVersionHeader, version.String())
	creds, err := s.mirrorHostCredentials()
	if err != nil {
		return 0, nil, endpointURL, fmt.Errorf("failed to determine request credentials: %s", err)
	}
	if creds != nil {
		// Note that if the initial requests gets redirected elsewhere
		// then the credentials will still be included in the new request,
		// even if they are on a different hostname. This is intentional
		// and consistent with how we handle credentials for other
		// Terraform-native services, because the user model is to configure
		// credentials for the "friendly hostname" they configured, not for
		// whatever hostname ends up ultimately serving the request as an
		// implementation detail.
		creds.PrepareRequest(req.Request)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, nil, endpointURL, err
	}
	defer func() {
		// If we're not returning the body then we'll close it
		// before we return.
		if body == nil {
			resp.Body.Close()
		}
	}()
	// After this point, our final URL return value should always be the
	// one from resp.Request, because that takes into account any redirects
	// we followed along the way.
	finalURL = resp.Request.URL

	if resp.StatusCode == http.StatusOK {
		// If and only if we get an OK response, we'll check that the response
		// type is JSON and return the body reader.
		ct := resp.Header.Get("Content-Type")
		mt, params, err := mime.ParseMediaType(ct)
		if err != nil {
			return 0, nil, finalURL, fmt.Errorf("response has invalid Content-Type: %s", err)
		}
		if mt != "application/json" {
			return 0, nil, finalURL, fmt.Errorf("response has invalid Content-Type: must be application/json")
		}
		for name := range params {
			// The application/json content-type has no defined parameters,
			// but some servers are configured to include a redundant "charset"
			// parameter anyway, presumably out of a sense of completeness.
			// We'll ignore them but warn that we're ignoring them in case the
			// subsequent parsing fails due to the server trying to use an
			// unsupported character encoding. (RFC 7159 defines its own
			// JSON-specific character encoding rules.)
			log.Printf("[WARN] Network mirror returned %q as part of its JSON content type, which is not defined. Ignoring.", name)
		}
		body = resp.Body
	}

	return resp.StatusCode, body, finalURL, nil
}

func (s *HTTPMirrorSource) errQueryFailed(provider addrs.Provider, err error) error {
	return ErrQueryFailed{
		Provider:  provider,
		Wrapped:   err,
		MirrorURL: s.baseURL,
	}
}

func (s *HTTPMirrorSource) errUnauthorized(finalURL *url.URL) error {
	hostname, err := svchostFromURL(finalURL)
	if err != nil {
		// Again, weird but we'll tolerate it.
		return fmt.Errorf("invalid credentials for %s", finalURL)
	}

	return ErrUnauthorized{
		Hostname: hostname,

		// We can't easily tell from here whether we had credentials or
		// not, so for now we'll just assume we did because "host rejected
		// the given credentials" is, hopefully, still understandable in
		// the event that there were none. (If this ends up being confusing
		// in practice then we'll need to do some refactoring of how
		// we handle credentials in this source.)
		HaveCredentials: true,
	}
}

func svchostFromURL(u *url.URL) (svchost.Hostname, error) {
	raw := u.Host

	// When "friendly hostnames" appear in Terraform-specific identifiers we
	// typically constrain their syntax more strictly than the
	// Internationalized Domain Name specifications call for, such as
	// forbidding direct use of punycode, but in this case we're just
	// working with a standard http: or https: URL and so we'll first use the
	// IDNA "lookup" rules directly, with no additional notational constraints,
	// to effectively normalize away the differences that would normally
	// produce an error.
	var portPortion string
	if colonPos := strings.Index(raw, ":"); colonPos != -1 {
		raw, portPortion = raw[:colonPos], raw[colonPos:]
	}
	// HTTPMirrorSource requires all URLs to be https URLs, because running
	// a network mirror over HTTP would potentially transmit any configured
	// credentials in cleartext. Therefore we don't need to do any special
	// handling of default ports here, because svchost.Hostname already
	// considers the absense of a port to represent the standard HTTPS port
	// 443, and will normalize away an explicit specification of port 443
	// in svchost.ForComparison below.

	normalized, err := idna.Display.ToUnicode(raw)
	if err != nil {
		return svchost.Hostname(""), err
	}

	// If ToUnicode succeeded above then "normalized" is now a hostname in the
	// normalized IDNA form, with any direct punycode already interpreted and
	// the case folding and other normalization rules applied. It should
	// therefore now be accepted by svchost.ForComparison with no additional
	// errors, but the port portion can still potentially be invalid.
	return svchost.ForComparison(normalized + portPortion)
}
