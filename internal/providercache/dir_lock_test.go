// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package providercache

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
)

func TestDirLockProviderVersion_serializesSamePackage(t *testing.T) {
	cacheDir := t.TempDir()
	platform := getproviders.Platform{OS: "linux", Arch: "amd64"}
	provider := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/aws")
	version := getproviders.MustParseVersion("5.100.0")

	first := NewDirWithPlatform(cacheDir, platform)
	unlockFirst, err := first.lockProviderVersion(context.Background(), provider, version)
	if err != nil {
		t.Fatalf("failed to acquire first provider lock: %s", err)
	}
	t.Cleanup(func() {
		_ = unlockFirst()
	})

	lockPath := getproviders.UnpackedDirectoryPathForPackage(cacheDir, provider, version, platform) + ".lock"
	if _, err := os.Lstat(filepath.FromSlash(lockPath)); err != nil {
		t.Fatalf("lock file does not exist at package-specific path %q: %s", lockPath, err)
	}

	type lockResult struct {
		unlock func() error
		err    error
	}
	resultCh := make(chan lockResult, 1)
	second := NewDirWithPlatform(cacheDir, platform)
	go func() {
		unlock, err := second.lockProviderVersion(context.Background(), provider, version)
		resultCh <- lockResult{unlock: unlock, err: err}
	}()

	select {
	case result := <-resultCh:
		if result.err == nil {
			_ = result.unlock()
		}
		t.Fatalf("second lock returned before first lock was released: %v", result.err)
	case <-time.After(250 * time.Millisecond):
	}

	if err := unlockFirst(); err != nil {
		t.Fatalf("failed to release first provider lock: %s", err)
	}

	select {
	case result := <-resultCh:
		if result.err != nil {
			t.Fatalf("second lock failed after first lock was released: %s", result.err)
		}
		if err := result.unlock(); err != nil {
			t.Fatalf("failed to release second provider lock: %s", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("second lock did not acquire after first lock was released")
	}
}

func TestDirLockProviderVersion_differentPackagesDoNotBlock(t *testing.T) {
	cacheDir := t.TempDir()
	platform := getproviders.Platform{OS: "linux", Arch: "amd64"}
	version := getproviders.MustParseVersion("5.100.0")
	awsProvider := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/aws")
	googleProvider := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/google")

	awsDir := NewDirWithPlatform(cacheDir, platform)
	unlockAWS, err := awsDir.lockProviderVersion(context.Background(), awsProvider, version)
	if err != nil {
		t.Fatalf("failed to acquire AWS provider lock: %s", err)
	}
	defer func() {
		_ = unlockAWS()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	googleDir := NewDirWithPlatform(cacheDir, platform)
	unlockGoogle, err := googleDir.lockProviderVersion(ctx, googleProvider, version)
	if err != nil {
		t.Fatalf("Google provider lock was blocked by AWS provider lock: %s", err)
	}
	if err := unlockGoogle(); err != nil {
		t.Fatalf("failed to release Google provider lock: %s", err)
	}
}

func TestDirLockProviderVersion_waitHonorsCancellation(t *testing.T) {
	cacheDir := t.TempDir()
	platform := getproviders.Platform{OS: "linux", Arch: "amd64"}
	provider := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/aws")
	version := getproviders.MustParseVersion("5.100.0")

	first := NewDirWithPlatform(cacheDir, platform)
	unlockFirst, err := first.lockProviderVersion(context.Background(), provider, version)
	if err != nil {
		t.Fatalf("failed to acquire first provider lock: %s", err)
	}
	defer func() {
		_ = unlockFirst()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan error, 1)
	second := NewDirWithPlatform(cacheDir, platform)
	go func() {
		_, err := second.lockProviderVersion(ctx, provider, version)
		resultCh <- err
	}()

	select {
	case err := <-resultCh:
		t.Fatalf("waiting lock returned before cancellation: %v", err)
	case <-time.After(250 * time.Millisecond):
	}

	cancel()
	select {
	case err := <-resultCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("waiting lock error = %v; want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("waiting lock did not return after cancellation")
	}
}
