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
	"time"

	"github.com/hashicorp/go-getter"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/internal/releaseauth"
)

// VersionManager downloads, caches, and returns information about versions
// of terraform-cloudplugin binaries downloaded from the specified backend.
type VersionManager struct {
	signingKey         string
	binaryName         string
	cloudPluginDataDir string
	host               svchost.Hostname
	client             *CloudPluginClient
	goos               string
	arch               string
	ctx                context.Context
}

// Version is a struct containing the path to the binary corresponding to
// the manifest version.
type Version struct {
	BinaryLocation    string
	ProductVersion    string
	ResolvedFromCache bool
}

const (
	KB = 1000
	MB = 1000 * KB
)

// NewVersionManager initializes a new VersionManager to broker data between the
// specified directory location containing cloudplugin package data and a
// Terraform Cloud backend URL.
func NewVersionManager(ctx context.Context, cloudPluginDataDir string, serviceURL *url.URL, goos, arch string) (*VersionManager, error) {
	client, err := NewCloudPluginClient(ctx, serviceURL)
	if err != nil {
		return nil, fmt.Errorf("could not initialize cloudplugin version manager: %w", err)
	}

	return &VersionManager{
		cloudPluginDataDir: cloudPluginDataDir,
		host:               svchost.Hostname(serviceURL.Host),
		client:             client,
		binaryName:         "terraform-cloudplugin",
		goos:               goos,
		arch:               arch,
		ctx:                ctx,
	}, nil
}

func (v VersionManager) versionedPackageLocation(version string) string {
	return path.Join(v.cloudPluginDataDir, "bin", version, fmt.Sprintf("%s_%s", v.goos, v.arch))
}

func (v VersionManager) cachedVersion(version string) *string {
	binaryPath := path.Join(v.versionedPackageLocation(version), v.binaryName)
	if _, err := os.Stat(binaryPath); err != nil {
		return nil
	}
	return &binaryPath
}

// Resolve fetches, authenticates, and caches a plugin binary matching the specifications
// and returns its location and version.
func (v VersionManager) Resolve() (*Version, error) {
	manifest, err := v.latestManifest(v.ctx)
	if err != nil {
		return nil, fmt.Errorf("could not resolve cloudplugin version for host %q: %w", v.host.ForDisplay(), err)
	}

	buildInfo, err := manifest.Select(v.goos, v.arch)
	if err != nil {
		return nil, err
	}

	// Check if there's a cached binary
	if cachedBinary := v.cachedVersion(manifest.ProductVersion); cachedBinary != nil {
		return &Version{
			BinaryLocation:    *cachedBinary,
			ProductVersion:    manifest.ProductVersion,
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
		return nil, fmt.Errorf("could not resolve cloudplugin version %q: %w", manifest.ProductVersion, err)
	}

	// Unarchive
	unzip := getter.ZipDecompressor{
		FilesLimit:    1,
		FileSizeLimit: 500 * MB,
	}
	targetPath := v.versionedPackageLocation(manifest.ProductVersion)
	log.Printf("[TRACE] decompressing %q to %q", t.Name(), targetPath)

	err = unzip.Decompress(targetPath, t.Name(), true, 0000)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress cloud plugin: %w", err)
	}

	return &Version{
		BinaryLocation:    path.Join(targetPath, v.binaryName),
		ProductVersion:    manifest.ProductVersion,
		ResolvedFromCache: false,
	}, nil
}

// Useful for small files that can be decoded all at once
func (v VersionManager) downloadFileBuffer(pathOrURL string) ([]byte, error) {
	buffer := bytes.Buffer{}
	err := v.client.DownloadFile(pathOrURL, &buffer)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), err
}

// verifyCloudPlugin authenticates the downloaded release archive
func (v VersionManager) verifyCloudPlugin(archiveManifest *Manifest, info *ManifestReleaseBuild, archiveLocation string) error {
	signature, err := v.downloadFileBuffer(archiveManifest.SHA256SumsSignatureURL)
	if err != nil {
		return fmt.Errorf("failed to download cloudplugin SHA256SUMS signature file: %w", err)
	}
	sums, err := v.downloadFileBuffer(archiveManifest.SHA256SumsURL)
	if err != nil {
		return fmt.Errorf("failed to download cloudplugin SHA256SUMS file: %w", err)
	}

	reportedSHA, err := releaseauth.SHA256FromHex(info.SHA256Sum)
	if err != nil {
		return fmt.Errorf("the reported checksum %q is not valid: %w", info.SHA256Sum, err)
	}
	checksums, err := releaseauth.ParseChecksums(sums)
	if err != nil {
		return fmt.Errorf("failed to parse cloudplugin SHA256SUMS file: %w", err)
	}

	sigAuth := releaseauth.NewSignatureAuthentication(signature, sums)
	if len(v.signingKey) > 0 {
		sigAuth.PublicKey = v.signingKey
	}

	all := releaseauth.AllAuthenticators(
		releaseauth.NewMatchingChecksumsAuthentication(reportedSHA, path.Base(info.URL), checksums),
		releaseauth.NewChecksumAuthentication(reportedSHA, archiveLocation),
		sigAuth,
	)

	return all.Authenticate()
}

func (v VersionManager) latestManifest(ctx context.Context) (*Manifest, error) {
	manifestCacheLocation := path.Join(v.cloudPluginDataDir, v.host.String(), "manifest.json")

	// Find the manifest cache for the hostname.
	info, err := os.Stat(manifestCacheLocation)
	modTime := time.Time{}
	var localManifest *Manifest
	if err != nil {
		log.Printf("[TRACE] no cloudplugin manifest cache found for host %q", v.host)
	} else {
		log.Printf("[TRACE] cloudplugin manifest cache found for host %q", v.host)
		modTime = info.ModTime()

		data, err := os.ReadFile(manifestCacheLocation)
		if err == nil {
			localManifest, err = decodeManifest(bytes.NewBuffer(data))
			if err != nil {
				log.Printf("[WARN] failed to decode cloudplugin manifest cache %q: %s", manifestCacheLocation, err)
			}
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
