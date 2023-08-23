package cloudplugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform/internal/httpclient"
	"github.com/hashicorp/terraform/internal/logging"
)

var (
	defaultRequestTimeout = 60 * time.Second
)

// ManifestReleaseBuild is the json-encoded details about a particular
// build of terraform-cloudplygin
type ManifestReleaseBuild struct {
	URL       string `json:"url"`
	SHA256Sum string `json:"sha256sum"`
}

// Manifest is the json-encoded manifest details sent by Terraform Cloud
type Manifest struct {
	ProductVersion         string                          `json:"plugin_version"`
	Archives               map[string]ManifestReleaseBuild `json:"archives"`
	SHA256SumsURL          string                          `json:"sha256sums_url"`
	SHA256SumsSignatureURL string                          `json:"sha256sums_signature_url"`
	lastModified           time.Time
}

// CloudPluginClient fetches and verifies release distributions of the cloudplugin
// that correspond to an upstream backend.
type CloudPluginClient struct {
	serviceURL *url.URL
	httpClient *retryablehttp.Client
	ctx        context.Context
}

func requestLogHook(logger retryablehttp.Logger, req *http.Request, i int) {
	if i > 0 {
		logger.Printf("[INFO] Previous request to the remote cloud manifest failed, attempting retry.")
	}
}

func decodeManifest(data io.Reader) (*Manifest, error) {
	var man Manifest
	dec := json.NewDecoder(data)
	if err := dec.Decode(&man); err != nil {
		return nil, ErrQueryFailed{
			inner: fmt.Errorf("failed to decode response body: %w", err),
		}
	}

	return &man, nil
}

// NewCloudPluginClient creates a new client for downloading and verifying
// terraform-cloudplugin archives
func NewCloudPluginClient(ctx context.Context, serviceURL *url.URL) (*CloudPluginClient, error) {
	httpClient := httpclient.New()
	httpClient.Timeout = defaultRequestTimeout

	retryableClient := retryablehttp.NewClient()
	retryableClient.HTTPClient = httpClient
	retryableClient.RetryMax = 3
	retryableClient.RequestLogHook = requestLogHook
	retryableClient.Logger = logging.HCLogger()

	return &CloudPluginClient{
		httpClient: retryableClient,
		serviceURL: serviceURL,
		ctx:        ctx,
	}, nil
}

// FetchManifest retrieves the cloudplugin manifest from Terraform Cloud,
// but returns a nil manifest if a 304 response is received, depending
// on the lastModified time.
func (c CloudPluginClient) FetchManifest(lastModified time.Time) (*Manifest, error) {
	req, _ := retryablehttp.NewRequestWithContext(c.ctx, "GET", c.serviceURL.JoinPath("manifest").String(), nil)
	req.Header.Set("If-Modified-Since", lastModified.Format(http.TimeFormat))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, ErrRequestCanceled
		}
		return nil, ErrQueryFailed{
			inner: err,
		}
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		lastModifiedRaw := resp.Header.Get("Last-Modified")
		if len(lastModifiedRaw) > 0 {
			lastModified, _ = time.Parse(http.TimeFormat, lastModifiedRaw)
		}
		manifest, err := decodeManifest(resp.Body)
		if err != nil {
			return nil, err
		}
		manifest.lastModified = lastModified
		return manifest, nil
	case http.StatusNotModified:
		return nil, nil
	case http.StatusNotFound:
		return nil, ErrCloudPluginNotSupported
	default:
		return nil, ErrQueryFailed{
			inner: errors.New(resp.Status),
		}
	}
}

// DownloadFile gets the URL at the specified path or URL and writes the
// contents to the specified Writer.
func (c CloudPluginClient) DownloadFile(pathOrURL string, writer io.Writer) error {
	url, err := c.resolveManifestURL(pathOrURL)
	if err != nil {
		return err
	}
	req, err := retryablehttp.NewRequestWithContext(c.ctx, "GET", url.String(), nil)
	if err != nil {
		return fmt.Errorf("invalid URL %q was provided by the cloudplugin manifest: %w", url, err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return ErrRequestCanceled
		}
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// OK
	case http.StatusNotFound:
		return ErrCloudPluginNotFound
	default:
		return ErrQueryFailed{
			inner: errors.New(resp.Status),
		}
	}

	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write downloaded file: %w", err)
	}

	return nil
}

func (c CloudPluginClient) resolveManifestURL(pathOrURL string) (*url.URL, error) {
	if strings.HasPrefix(pathOrURL, "/") {
		copy := *c.serviceURL
		copy.Path = ""
		return copy.JoinPath(pathOrURL), nil
	}

	result, err := url.Parse(pathOrURL)
	if err != nil {
		return nil, fmt.Errorf("received malformed URL %q from cloudplugin manifest: %w", pathOrURL, err)
	}
	return result, nil
}

// Select gets the specific build data from the Manifest for the specified OS/Architecture
func (m Manifest) Select(goos, arch string) (*ManifestReleaseBuild, error) {
	var supported []string
	for key := range m.Archives {
		supported = append(supported, key)
	}

	osArchKey := fmt.Sprintf("%s_%s", goos, arch)
	log.Printf("[TRACE] checking for cloudplugin archive for %s. Supported architectures: %v", osArchKey, supported)

	archiveOSArch, ok := m.Archives[osArchKey]
	if !ok {
		return nil, ErrArchNotSupported
	}

	return &archiveOSArch, nil
}
