// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

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
	"github.com/hashicorp/terraform/internal/releaseauth"
)

var (
	defaultRequestTimeout = 60 * time.Second
)

// SHASumsSignatures holds a list of URLs, each referring a detached signature of the release's build artifacts.
type SHASumsSignatures []string

// BuildArtifact represents a single build artifact in a release response.
type BuildArtifact struct {

	// The hardware architecture of the build artifact
	// Enum: [386 all amd64 amd64-lxc arm arm5 arm6 arm64 arm7 armelv5 armhfv6 i686 mips mips64 mipsle ppc64le s390x ui x86_64]
	Arch string `json:"arch"`

	// The Operating System corresponding to the build artifact
	// Enum: [archlinux centos darwin debian dragonfly freebsd linux netbsd openbsd plan9 python solaris terraform web windows]
	Os string `json:"os"`

	// This build is unsupported and provided for convenience only.
	Unsupported bool `json:"unsupported,omitempty"`

	// The URL where this build can be downloaded
	URL string `json:"url"`
}

// ReleaseStatus Status of the product release
// Example: {"message":"This release is supported","state":"supported"}
type ReleaseStatus struct {

	// Provides information about the most recent change; must be provided when Name="withdrawn"
	Message string `json:"message,omitempty"`

	// The state name of the release
	// Enum: [supported unsupported withdrawn]
	State string `json:"state"`

	// The timestamp for the creation of the product release status
	// Example: 2009-11-10T23:00:00Z
	// Format: date-time
	TimestampUpdated time.Time `json:"timestamp_updated"`
}

// Release All metadata for a single product release
type Release struct {
	// builds
	Builds []*BuildArtifact `json:"builds,omitempty"`

	// A docker image name and tag for this release in the format `name`:`tag`
	// Example: consul:1.10.0-beta3
	DockerNameTag string `json:"docker_name_tag,omitempty"`

	// True if and only if this product release is a prerelease.
	IsPrerelease bool `json:"is_prerelease"`

	// The license class indicates how this product is licensed.
	// Enum: [enterprise hcp oss]
	LicenseClass string `json:"license_class"`

	// The product name
	// Example: consul-enterprise
	// Required: true
	Name string `json:"name"`

	// Status
	Status ReleaseStatus `json:"status"`

	// Timestamp at which this product release was created.
	// Example: 2009-11-10T23:00:00Z
	// Format: date-time
	TimestampCreated time.Time `json:"timestamp_created"`

	// Timestamp when this product release was most recently updated.
	// Example: 2009-11-10T23:00:00Z
	// Format: date-time
	TimestampUpdated time.Time `json:"timestamp_updated"`

	// URL for a blogpost announcing this release
	URLBlogpost string `json:"url_blogpost,omitempty"`

	// URL for the changelog covering this release
	URLChangelog string `json:"url_changelog,omitempty"`

	// The project's docker repo on Amazon ECR-Public
	URLDockerRegistryDockerhub string `json:"url_docker_registry_dockerhub,omitempty"`

	// The project's docker repo on DockerHub
	URLDockerRegistryEcr string `json:"url_docker_registry_ecr,omitempty"`

	// URL for the software license applicable to this release
	// Required: true
	URLLicense string `json:"url_license,omitempty"`

	// The project's website URL
	URLProjectWebsite string `json:"url_project_website,omitempty"`

	// URL for this release's change notes
	URLReleaseNotes string `json:"url_release_notes,omitempty"`

	// URL for this release's file containing checksums of all the included build artifacts
	URLSHASums string `json:"url_shasums"`

	// An array of URLs, each pointing to a signature file.  Each signature file is a detached signature
	// of the checksums file (see field `url_shasums`).  Signature files may or may not embed the signing
	// key ID in the filename.
	URLSHASumsSignatures SHASumsSignatures `json:"url_shasums_signatures"`

	// URL for the product's source code repository.  This field is empty for
	// enterprise and hcp products.
	URLSourceRepository string `json:"url_source_repository,omitempty"`

	// The version of this release
	// Example: 1.10.0-beta3
	// Required: true
	Version string `json:"version"`
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

func decodeManifest(data io.Reader) (*Release, error) {
	var man Release
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

// FetchManifest retrieves the cloudplugin manifest from HCP Terraform,
// but returns a nil manifest if a 304 response is received, depending
// on the lastModified time.
func (c CloudPluginClient) FetchManifest(lastModified time.Time) (*Release, error) {
	req, _ := retryablehttp.NewRequestWithContext(c.ctx, "GET", c.serviceURL.JoinPath("manifest.json").String(), nil)
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
		manifest, err := decodeManifest(resp.Body)
		if err != nil {
			return nil, err
		}
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
func (m Release) Select(goos, arch string) (*BuildArtifact, error) {
	var supported []string
	var found *BuildArtifact
	for _, build := range m.Builds {
		key := fmt.Sprintf("%s_%s", build.Os, build.Arch)
		supported = append(supported, key)

		if goos == build.Os && arch == build.Arch {
			found = build
		}
	}

	osArchKey := fmt.Sprintf("%s_%s", goos, arch)
	log.Printf("[TRACE] checking for cloudplugin archive for %s. Supported architectures: %v", osArchKey, supported)

	if found == nil {
		return nil, ErrArchNotSupported
	}

	return found, nil
}

// PrimarySHASumsSignatureURL returns the URL among the URLSHASumsSignatures that matches
// the public key known by this version of terraform. It falls back to the first URL with no
// ID in the URL.
func (m Release) PrimarySHASumsSignatureURL() (string, error) {
	if len(m.URLSHASumsSignatures) == 0 {
		return "", fmt.Errorf("no SHA256SUMS URLs were available")
	}

	findBySuffix := func(suffix string) string {
		for _, url := range m.URLSHASumsSignatures {
			if len(url) > len(suffix) && strings.EqualFold(suffix, url[len(url)-len(suffix):]) {
				return url
			}
		}
		return ""
	}

	withKeyID := findBySuffix(fmt.Sprintf(".%s.sig", releaseauth.HashiCorpPublicKeyID))
	if withKeyID == "" {
		withNoKeyID := findBySuffix("_SHA256SUMS.sig")
		if withNoKeyID == "" {
			return "", fmt.Errorf("no SHA256SUMS URLs matched the known public key")
		}
		return withNoKeyID, nil
	}
	return withKeyID, nil
}
