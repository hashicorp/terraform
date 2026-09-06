// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package providercache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/replacefile"
)

const providerLockRetryInterval = 100 * time.Millisecond

const providerCacheMetadataVersion = 1

type providerCacheMetadata struct {
	Version      int                 `json:"version"`
	PackageHash  getproviders.Hash   `json:"package_hash"`
	SignedHashes []getproviders.Hash `json:"signed_hashes,omitempty"`
}

func (d *Dir) providerVersionPath(provider addrs.Provider, version getproviders.Version) string {
	return getproviders.UnpackedDirectoryPathForPackage(
		d.baseDir, provider, version, d.targetPlatform,
	)
}

// lockProviderVersion acquires an advisory lock for one exact provider package
// in this cache directory. The lock file remains after unlocking so all
// contenders continue to coordinate through the same filesystem object.
func (d *Dir) lockProviderVersion(ctx context.Context, provider addrs.Provider, version getproviders.Version) (func() error, error) {
	packagePath := d.providerVersionPath(provider, version)
	lockPath := packagePath + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create provider cache lock directory: %w", err)
	}

	log.Printf("[TRACE] providercache.Dir: waiting for lock on %s v%s", provider, version)
	fileLock := flock.New(filepath.FromSlash(lockPath))
	locked, err := fileLock.TryLockContext(ctx, providerLockRetryInterval)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire provider cache lock for %s v%s: %w", provider, version, err)
	}
	if !locked {
		return nil, fmt.Errorf("failed to acquire provider cache lock for %s v%s", provider, version)
	}
	log.Printf("[TRACE] providercache.Dir: acquired lock on %s v%s", provider, version)

	// Another process might have populated this package while we waited.
	d.metaCache = nil

	return func() error {
		log.Printf("[TRACE] providercache.Dir: releasing lock on %s v%s", provider, version)
		return fileLock.Unlock()
	}, nil
}

// readProviderCacheMetadata returns the completion metadata written by a
// successful installer. Callers must hold the corresponding provider lock.
func (d *Dir) readProviderCacheMetadata(provider addrs.Provider, version getproviders.Version) (*providerCacheMetadata, error) {
	metadataPath := d.providerVersionPath(provider, version) + ".metadata.json"
	src, err := os.ReadFile(filepath.FromSlash(metadataPath))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read provider cache metadata: %w", err)
	}

	var metadata providerCacheMetadata
	if err := json.Unmarshal(src, &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode provider cache metadata: %w", err)
	}
	if metadata.Version != providerCacheMetadataVersion || metadata.PackageHash == "" {
		return nil, fmt.Errorf("provider cache metadata has an unsupported or incomplete format")
	}
	return &metadata, nil
}

// writeProviderCacheMetadata publishes a successful install's verified hash
// information atomically. Callers must hold the corresponding provider lock.
func (d *Dir) writeProviderCacheMetadata(provider addrs.Provider, version getproviders.Version, packageHash getproviders.Hash, signedHashes []getproviders.Hash) error {
	metadata := providerCacheMetadata{
		Version:      providerCacheMetadataVersion,
		PackageHash:  packageHash,
		SignedHashes: signedHashes,
	}
	src, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to encode provider cache metadata: %w", err)
	}

	metadataPath := filepath.FromSlash(d.providerVersionPath(provider, version) + ".metadata.json")
	if err := replacefile.AtomicWriteFile(metadataPath, src, 0600); err != nil {
		return fmt.Errorf("failed to write provider cache metadata: %w", err)
	}
	return nil
}
