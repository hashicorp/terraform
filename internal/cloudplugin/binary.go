// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloudplugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-getter"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/internal/releaseauth"
)

// BinaryManager downloads, caches, and returns information about the
// terraform-cloudplugin binary downloaded from the specified backend.
type BinaryManager struct {
	signingKey         string
	binaryName         string
	cloudPluginDataDir string
	overridePath       string
	host               svchost.Hostname
	client             *CloudPluginClient
	goos               string
	arch               string
	ctx                context.Context
}

// Binary is a struct containing the path to an authenticated binary corresponding to
// a backend service.
type Binary struct {
	Path                    string
	ProductVersion          string
	ResolvedFromCache       bool
	ResolvedFromDevOverride bool
}

const (
	KB = 1000
	MB = 1000 * KB
)

// BinaryManager initializes a new BinaryManager to broker data between the
// specified directory location containing cloudplugin package data and a
// HCP Terraform backend URL.
func NewBinaryManager(ctx context.Context, cloudPluginDataDir, overridePath string, serviceURL *url.URL, goos, arch string) (*BinaryManager, error) {
	client, err := NewCloudPluginClient(ctx, serviceURL)
	if err != nil {
		return nil, fmt.Errorf("could not initialize cloudplugin version manager: %w", err)
	}

	return &BinaryManager{
		cloudPluginDataDir: cloudPluginDataDir,
		overridePath:       overridePath,
		host:               svchost.Hostname(serviceURL.Host),
		client:             client,
		binaryName:         "terraform-cloudplugin",
		goos:               goos,
		arch:               arch,
		ctx:                ctx,
	}, nil
}

func (v BinaryManager) binaryLocation() string {
	return path.Join(v.cloudPluginDataDir, "bin", fmt.Sprintf("%s_%s", v.goos, v.arch))
}

func (v BinaryManager) cachedVersion(version string) *string {
	binaryPath := path.Join(v.binaryLocation(), v.binaryName)

	if _, err := os.Stat(binaryPath); err != nil {
		return nil
	}

	// The version from the manifest must match the contents of ".version"
	versionData, err := os.ReadFile(path.Join(v.binaryLocation(), ".version"))
	if err != nil || strings.Trim(string(versionData), " \n\r\t") != version {
		return nil
	}

	return &binaryPath
}

// Resolve fetches, authenticates, and caches a plugin binary matching the specifications
// and returns its location and version.
func (v BinaryManager) Resolve() (*Binary, error) {
	if v.overridePath != "" {
		log.Printf("[TRACE] Using dev override for cloudplugin binary")
		return v.resolveDev()
	}
	return v.resolveRelease()
}

func (v BinaryManager) resolveDev() (*Binary, error) {
	return &Binary{
		Path:                    v.overridePath,
		ProductVersion:          "dev",
		ResolvedFromDevOverride: true,
	}, nil
}

func (v BinaryManager) resolveRelease() (*Binary, error) {
	manifest, err := v.latestManifest(v.ctx)
	if err != nil {
		return nil, fmt.Errorf("could not resolve cloudplugin version for host %q: %w", v.host.ForDisplay(), err)
	}

	buildInfo, err := manifest.Select(v.goos, v.arch)
	if err != nil {
		return nil, err
	}

	// Check if there's a cached binary
	if cachedBinary := v.cachedVersion(manifest.Version); cachedBinary != nil {
		return &Binary{
			Path:              *cachedBinary,
			ProductVersion:    manifest.Version,
			ResolvedFromCache: true,
		}, nil
	}

	// Download the archive
	t, err := os.CreateTemp(os.TempDir(), "terraform-cloudplugin")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file for download: %w", err)
	}
	defer os.Remove(t.Name())

	err = v.client.DownloadFile(buildInfo.URL, t)
	if err != nil {
		return nil, err
	}
	t.Close() // Close only returns an error if it's already been called

	// Authenticate the archive
	err = v.verifyCloudPlugin(manifest, buildInfo, t.Name())
	if err != nil {
		return nil, fmt.Errorf("could not resolve cloudplugin version %q: %w", manifest.Version, err)
	}

	// Unarchive
	unzip := getter.ZipDecompressor{
		FilesLimit:    1,
		FileSizeLimit: 500 * MB,
	}
	targetPath := v.binaryLocation()
	log.Printf("[TRACE] decompressing %q to %q", t.Name(), targetPath)

	err = unzip.Decompress(targetPath, t.Name(), true, 0000)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress cloud plugin: %w", err)
	}

	err = os.WriteFile(path.Join(targetPath, ".version"), []byte(manifest.Version), 0644)
	if err != nil {
		log.Printf("[ERROR] failed to write .version file to %q: %s", targetPath, err)
	}

	return &Binary{
		Path:              path.Join(targetPath, v.binaryName),
		ProductVersion:    manifest.Version,
		ResolvedFromCache: false,
	}, nil
}

// Useful for small files that can be decoded all at once
func (v BinaryManager) downloadFileBuffer(pathOrURL string) ([]byte, error) {
	buffer := bytes.Buffer{}
	err := v.client.DownloadFile(pathOrURL, &buffer)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), err
}

// verifyCloudPlugin authenticates the downloaded release archive
func (v BinaryManager) verifyCloudPlugin(archiveManifest *Release, info *BuildArtifact, archiveLocation string) error {
	signature, err := v.downloadFileBuffer(archiveManifest.URLSHASumsSignatures[0])
	if err != nil {
		return fmt.Errorf("failed to download cloudplugin SHA256SUMS signature file: %w", err)
	}
	sums, err := v.downloadFileBuffer(archiveManifest.URLSHASums)
	if err != nil {
		return fmt.Errorf("failed to download cloudplugin SHA256SUMS file: %w", err)
	}

	checksums, err := releaseauth.ParseChecksums(sums)
	if err != nil {
		return fmt.Errorf("failed to parse cloudplugin SHA256SUMS file: %w", err)
	}

	filename := path.Base(info.URL)
	reportedSHA, ok := checksums[filename]
	if !ok {
		return fmt.Errorf("could not find checksum for file %q", filename)
	}

	sigAuth := releaseauth.NewSignatureAuthentication(signature, sums)
	if len(v.signingKey) > 0 {
		sigAuth.PublicKey = v.signingKey
	}

	all := releaseauth.AllAuthenticators(
		releaseauth.NewChecksumAuthentication(reportedSHA, archiveLocation),
		sigAuth,
	)

	return all.Authenticate()
}

func (v BinaryManager) latestManifest(ctx context.Context) (*Release, error) {
	manifestCacheLocation := path.Join(v.cloudPluginDataDir, v.host.String(), "manifest.json")

	// Find the manifest cache for the hostname.
	data, err := os.ReadFile(manifestCacheLocation)
	modTime := time.Time{}
	var localManifest *Release
	if err != nil {
		log.Printf("[TRACE] no cloudplugin manifest cache found for host %q", v.host)
	} else {
		log.Printf("[TRACE] cloudplugin manifest cache found for host %q", v.host)

		localManifest, err = decodeManifest(bytes.NewBuffer(data))
		modTime = localManifest.TimestampUpdated
		if err != nil {
			log.Printf("[WARN] failed to decode cloudplugin manifest cache %q: %s", manifestCacheLocation, err)
		}
	}

	// Even though we may have a local manifest, always see if there is a newer remote manifest
	result, err := v.client.FetchManifest(modTime)
	// FetchManifest can return nil, nil (see below)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cloudplugin manifest: %w", err)
	}

	// No error and no remoteManifest means the existing manifest is not modified
	// and it's safe to use the local manifest
	if result == nil && localManifest != nil {
		result = localManifest
	} else {
		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to dump cloudplugin manifest to JSON: %w", err)
		}

		// Ensure target directory exists
		if err := os.MkdirAll(filepath.Dir(manifestCacheLocation), 0755); err != nil {
			return nil, fmt.Errorf("failed to create cloudplugin manifest cache directory: %w", err)
		}

		output, err := os.Create(manifestCacheLocation)
		if err != nil {
			return nil, fmt.Errorf("failed to create cloudplugin manifest cache: %w", err)
		}

		_, err = output.Write(data)
		if err != nil {
			return nil, fmt.Errorf("failed to write cloudplugin manifest cache: %w", err)
		}
		log.Printf("[TRACE] wrote cloudplugin manifest cache to %q", manifestCacheLocation)
	}

	return result, nil
}
