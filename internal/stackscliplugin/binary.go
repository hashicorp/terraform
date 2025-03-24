// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackscliplugin

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

// StacksCLIBinaryManager downloads, caches, and returns information about the
// stacks-cli plugin binary downloaded from the specified backend.
type StacksCLIBinaryManager struct {
	signingKey             string
	binaryName             string
	stacksCLIPluginDataDir string
	overridePath           string
	host                   svchost.Hostname
	client                 *StacksCLIPluginClient
	goos                   string
	arch                   string
	ctx                    context.Context
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

const binaryName = "terraform-stacks-cli-plugin"

// StacksCLIBinaryManager initializes a new StacksCLIBinaryManager to broker data between the
// specified directory location containing stacks-cli plugin package data and a
// HCP Terraform backend URL.
func NewStacksCLIBinaryManager(ctx context.Context, stacksCLIPluginDataDir, overridePath string, serviceURL *url.URL, goos, arch string) (*StacksCLIBinaryManager, error) {
	client, err := NewStacksCLIPluginClient(ctx, serviceURL)
	if err != nil {
		return nil, fmt.Errorf("could not initialize stacks-cli plugin version manager: %w", err)
	}

	return &StacksCLIBinaryManager{
		stacksCLIPluginDataDir: stacksCLIPluginDataDir,
		overridePath:           overridePath,
		host:                   svchost.Hostname(serviceURL.Host),
		client:                 client,
		binaryName:             binaryName,
		goos:                   goos,
		arch:                   arch,
		ctx:                    ctx,
	}, nil
}

func (v StacksCLIBinaryManager) binaryLocation() string {
	return path.Join(v.stacksCLIPluginDataDir, "bin", fmt.Sprintf("%s_%s", v.goos, v.arch))
}

func (v StacksCLIBinaryManager) cachedVersion(version string) *string {
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
func (v StacksCLIBinaryManager) Resolve() (*Binary, error) {
	if v.overridePath != "" {
		log.Printf("[TRACE] Using dev override for stacks-cli plugin binary")
		return v.resolveDev()
	}
	return v.resolveRelease()
}

func (v StacksCLIBinaryManager) resolveDev() (*Binary, error) {
	return &Binary{
		Path:                    v.overridePath,
		ProductVersion:          "dev",
		ResolvedFromDevOverride: true,
	}, nil
}

func (v StacksCLIBinaryManager) resolveRelease() (*Binary, error) {
	manifest, err := v.latestManifest(v.ctx)
	if err != nil {
		return nil, fmt.Errorf("could not resolve stacks-cli plugin version for host %q: %w", v.host.ForDisplay(), err)
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
	t, err := os.CreateTemp(os.TempDir(), binaryName)
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
	err = v.verifyStacksCLIPlugin(manifest, buildInfo, t.Name())
	if err != nil {
		return nil, fmt.Errorf("could not resolve stacks-cli plugin version %q: %w", manifest.Version, err)
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
		return nil, fmt.Errorf("failed to decompress stacks-cli plugin: %w", err)
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
func (v StacksCLIBinaryManager) downloadFileBuffer(pathOrURL string) ([]byte, error) {
	buffer := bytes.Buffer{}
	err := v.client.DownloadFile(pathOrURL, &buffer)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), err
}

// verifyStacksCLIPlugin authenticates the downloaded release archive
func (v StacksCLIBinaryManager) verifyStacksCLIPlugin(archiveManifest *Release, info *BuildArtifact, archiveLocation string) error {
	signature, err := v.downloadFileBuffer(archiveManifest.URLSHASumsSignatures[0])
	if err != nil {
		return fmt.Errorf("failed to download stacks-cli plugin SHA256SUMS signature file: %w", err)
	}
	sums, err := v.downloadFileBuffer(archiveManifest.URLSHASums)
	if err != nil {
		return fmt.Errorf("failed to download stacks-cli plugin SHA256SUMS file: %w", err)
	}

	checksums, err := releaseauth.ParseChecksums(sums)
	if err != nil {
		return fmt.Errorf("failed to parse stacks-cli plugin SHA256SUMS file: %w", err)
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

func (v StacksCLIBinaryManager) latestManifest(ctx context.Context) (*Release, error) {
	manifestCacheLocation := path.Join(v.stacksCLIPluginDataDir, v.host.String(), "manifest.json")

	// Find the manifest cache for the hostname.
	data, err := os.ReadFile(manifestCacheLocation)
	modTime := time.Time{}
	var localManifest *Release
	if err != nil {
		log.Printf("[TRACE] no stacks-cli plugin manifest cache found for host %q", v.host)
	} else {
		log.Printf("[TRACE] stacks-cli plugin manifest cache found for host %q", v.host)

		localManifest, err = decodeManifest(bytes.NewBuffer(data))
		modTime = localManifest.TimestampUpdated
		if err != nil {
			log.Printf("[WARN] failed to decode stacks-cli plugin manifest cache %q: %s", manifestCacheLocation, err)
		}
	}

	// Even though we may have a local manifest, always see if there is a newer remote manifest
	result, err := v.client.FetchManifest(modTime)
	// FetchManifest can return nil, nil (see below)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stacks-cli plugin manifest: %w", err)
	}

	// No error and no remoteManifest means the existing manifest is not modified
	// and it's safe to use the local manifest
	if result == nil && localManifest != nil {
		result = localManifest
	} else {
		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to dump stacks-cli plugin manifest to JSON: %w", err)
		}

		// Ensure target directory exists
		if err := os.MkdirAll(filepath.Dir(manifestCacheLocation), 0755); err != nil {
			return nil, fmt.Errorf("failed to create stacks-cli plugin manifest cache directory: %w", err)
		}

		output, err := os.Create(manifestCacheLocation)
		if err != nil {
			return nil, fmt.Errorf("failed to create stacks-cli plugin manifest cache: %w", err)
		}

		_, err = output.Write(data)
		if err != nil {
			return nil, fmt.Errorf("failed to write stacks-cli plugin manifest cache: %w", err)
		}
		log.Printf("[TRACE] wrote stacks-cli plugin manifest cache to %q", manifestCacheLocation)
	}

	return result, nil
}
