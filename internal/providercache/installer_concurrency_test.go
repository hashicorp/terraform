// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package providercache

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
)

func TestEnsureProviderVersionsConcurrent_samePackage(t *testing.T) {
	provider := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/aws")
	version := getproviders.MustParseVersion("5.100.0")
	constraints := getproviders.MustParseVersionConstraints("= 5.100.0")
	platform := getproviders.CurrentPlatform

	archive, checksum, err := getproviders.CreateFakeFileWithChecksumForProvider(
		t, provider, version, platform, "",
	)
	if err != nil {
		t.Fatalf("failed to create provider archive: %s", err)
	}
	archiveBytes, err := os.ReadFile(archive.Name())
	if err != nil {
		t.Fatalf("failed to read provider archive: %s", err)
	}

	var downloadCount atomic.Int32
	firstRequestStarted := make(chan struct{})
	secondRequestStarted := make(chan struct{})
	releaseDownload := make(chan struct{})
	var signalFirst sync.Once
	var releaseOnce sync.Once
	releaseDownloads := func() {
		releaseOnce.Do(func() {
			close(releaseDownload)
		})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestNumber := downloadCount.Add(1)
		signalFirst.Do(func() {
			close(firstRequestStarted)
		})
		if requestNumber == 2 {
			close(secondRequestStarted)
		}
		<-releaseDownload
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(archiveBytes)
	}))
	defer server.Close()
	defer releaseDownloads()

	meta := getproviders.PackageMeta{
		Provider:         provider,
		Version:          version,
		ProtocolVersions: getproviders.VersionList{getproviders.MustParseVersion("5.0.0")},
		TargetPlatform:   platform,
		Filename:         filepath.Base(archive.Name()),
		Location:         getproviders.PackageHTTPURL(server.URL + "/terraform-provider-aws.zip"),
		Authentication:   getproviders.NewArchiveChecksumAuthentication(platform, checksum),
	}
	reqs := getproviders.Requirements{
		provider: constraints,
	}
	newLocks := func() *depsfile.Locks {
		return depsfile.NewLocks()
	}

	globalCachePath := t.TempDir()
	firstTarget := NewDirWithPlatform(t.TempDir(), platform)
	secondTarget := NewDirWithPlatform(t.TempDir(), platform)
	firstInstaller := NewInstaller(firstTarget, getproviders.NewMockSource([]getproviders.PackageMeta{meta}, nil))
	firstInstaller.SetGlobalCacheDir(NewDirWithPlatform(globalCachePath, platform))
	secondInstaller := NewInstaller(secondTarget, getproviders.NewMockSource([]getproviders.PackageMeta{meta}, nil))
	secondInstaller.SetGlobalCacheDir(NewDirWithPlatform(globalCachePath, platform))

	type installResult struct {
		locks *depsfile.Locks
		err   error
	}
	firstResult := make(chan installResult, 1)
	secondResult := make(chan installResult, 1)
	go func() {
		locks, err := firstInstaller.EnsureProviderVersions(
			context.Background(), newLocks(), reqs, InstallNewProvidersOnly,
		)
		firstResult <- installResult{locks: locks, err: err}
	}()

	select {
	case <-firstRequestStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("first provider download did not start")
	}

	go func() {
		locks, err := secondInstaller.EnsureProviderVersions(
			context.Background(), newLocks(), reqs, InstallNewProvidersOnly,
		)
		secondResult <- installResult{locks: locks, err: err}
	}()

	// On the unsafe implementation the second download starts while the first
	// is blocked. With package locking it waits for the first to complete.
	select {
	case <-secondRequestStarted:
	case <-time.After(500 * time.Millisecond):
	}
	releaseDownloads()

	for name, resultCh := range map[string]<-chan installResult{
		"first":  firstResult,
		"second": secondResult,
	} {
		select {
		case result := <-resultCh:
			if result.err != nil {
				t.Errorf("%s provider installation failed: %s", name, result.err)
			}
			if result.locks == nil || result.locks.Provider(provider) == nil {
				t.Errorf("%s provider installation returned no dependency lock", name)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s provider installation did not complete", name)
		}
	}

	if got, want := downloadCount.Load(), int32(1); got != want {
		t.Fatalf("provider archive download count = %d; want %d", got, want)
	}

	globalPackagePath := getproviders.UnpackedDirectoryPathForPackage(
		globalCachePath, provider, version, platform,
	)
	for name, target := range map[string]*Dir{
		"first":  firstTarget,
		"second": secondTarget,
	} {
		entry := target.ProviderVersion(provider, version)
		if entry == nil {
			t.Errorf("%s target has no provider cache entry", name)
			continue
		}
		if runtime.GOOS != "windows" {
			info, err := os.Lstat(filepath.FromSlash(entry.PackageDir))
			if err != nil {
				t.Errorf("%s target package cannot be inspected: %s", name, err)
				continue
			}
			if info.Mode()&os.ModeSymlink == 0 {
				t.Errorf("%s target package is not a symlink", name)
			}
		}
		resolved, err := filepath.EvalSymlinks(filepath.FromSlash(entry.PackageDir))
		if err != nil {
			t.Errorf("%s target package symlink cannot be resolved: %s", name, err)
			continue
		}
		wantResolved, err := filepath.EvalSymlinks(filepath.FromSlash(globalPackagePath))
		if err != nil {
			t.Fatalf("global package path cannot be resolved: %s", err)
		}
		if resolved != wantResolved {
			t.Errorf("%s target resolves to %q; want %q", name, resolved, wantResolved)
		}
	}
}

func TestEnsureProviderVersionsConcurrent_differentPackages(t *testing.T) {
	version := getproviders.MustParseVersion("5.100.0")
	constraints := getproviders.MustParseVersionConstraints("= 5.100.0")
	platform := getproviders.CurrentPlatform
	awsProvider := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/aws")
	googleProvider := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/google")

	awsArchive, awsChecksum, err := getproviders.CreateFakeFileWithChecksumForProvider(
		t, awsProvider, version, platform, "",
	)
	if err != nil {
		t.Fatalf("failed to create AWS provider archive: %s", err)
	}
	awsBytes, err := os.ReadFile(awsArchive.Name())
	if err != nil {
		t.Fatalf("failed to read AWS provider archive: %s", err)
	}
	googleArchive, googleChecksum, err := getproviders.CreateFakeFileWithChecksumForProvider(
		t, googleProvider, version, platform, "",
	)
	if err != nil {
		t.Fatalf("failed to create Google provider archive: %s", err)
	}
	googleBytes, err := os.ReadFile(googleArchive.Name())
	if err != nil {
		t.Fatalf("failed to read Google provider archive: %s", err)
	}

	awsStarted := make(chan struct{})
	googleStarted := make(chan struct{})
	releaseAWS := make(chan struct{})
	var signalAWS sync.Once
	var signalGoogle sync.Once
	var releaseOnce sync.Once
	releaseAWSDownload := func() {
		releaseOnce.Do(func() {
			close(releaseAWS)
		})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		switch r.URL.Path {
		case "/aws.zip":
			signalAWS.Do(func() {
				close(awsStarted)
			})
			<-releaseAWS
			_, _ = w.Write(awsBytes)
		case "/google.zip":
			signalGoogle.Do(func() {
				close(googleStarted)
			})
			_, _ = w.Write(googleBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	defer releaseAWSDownload()

	awsMeta := getproviders.PackageMeta{
		Provider:         awsProvider,
		Version:          version,
		ProtocolVersions: getproviders.VersionList{getproviders.MustParseVersion("5.0.0")},
		TargetPlatform:   platform,
		Filename:         filepath.Base(awsArchive.Name()),
		Location:         getproviders.PackageHTTPURL(server.URL + "/aws.zip"),
		Authentication:   getproviders.NewArchiveChecksumAuthentication(platform, awsChecksum),
	}
	googleMeta := getproviders.PackageMeta{
		Provider:         googleProvider,
		Version:          version,
		ProtocolVersions: getproviders.VersionList{getproviders.MustParseVersion("5.0.0")},
		TargetPlatform:   platform,
		Filename:         filepath.Base(googleArchive.Name()),
		Location:         getproviders.PackageHTTPURL(server.URL + "/google.zip"),
		Authentication:   getproviders.NewArchiveChecksumAuthentication(platform, googleChecksum),
	}

	globalCachePath := t.TempDir()
	runInstall := func(provider addrs.Provider, meta getproviders.PackageMeta) <-chan error {
		resultCh := make(chan error, 1)
		targetPath := t.TempDir()
		go func() {
			target := NewDirWithPlatform(targetPath, platform)
			installer := NewInstaller(target, getproviders.NewMockSource([]getproviders.PackageMeta{meta}, nil))
			installer.SetGlobalCacheDir(NewDirWithPlatform(globalCachePath, platform))
			_, err := installer.EnsureProviderVersions(
				context.Background(),
				depsfile.NewLocks(),
				getproviders.Requirements{provider: constraints},
				InstallNewProvidersOnly,
			)
			resultCh <- err
		}()
		return resultCh
	}

	awsResult := runInstall(awsProvider, awsMeta)
	select {
	case <-awsStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("AWS provider download did not start")
	}

	googleResult := runInstall(googleProvider, googleMeta)
	select {
	case <-googleStarted:
		// The Google package has a different lock identity and must not wait.
	case <-time.After(2 * time.Second):
		t.Fatal("Google provider download was blocked by AWS provider download")
	}
	releaseAWSDownload()

	for name, resultCh := range map[string]<-chan error{
		"AWS":    awsResult,
		"Google": googleResult,
	} {
		select {
		case err := <-resultCh:
			if err != nil {
				t.Errorf("%s provider installation failed: %s", name, err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s provider installation did not complete", name)
		}
	}
}

func TestEnsureProviderVersionsConcurrent_failedInstallReleasesLock(t *testing.T) {
	provider := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/aws")
	version := getproviders.MustParseVersion("5.100.0")
	constraints := getproviders.MustParseVersionConstraints("= 5.100.0")
	platform := getproviders.CurrentPlatform

	archive, checksum, err := getproviders.CreateFakeFileWithChecksumForProvider(
		t, provider, version, platform, "",
	)
	if err != nil {
		t.Fatalf("failed to create provider archive: %s", err)
	}
	archiveBytes, err := os.ReadFile(archive.Name())
	if err != nil {
		t.Fatalf("failed to read provider archive: %s", err)
	}
	packageHash, err := getproviders.PackageHashV1(getproviders.PackageLocalArchive(archive.Name()))
	if err != nil {
		t.Fatalf("failed to calculate provider package hash: %s", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		switch r.URL.Path {
		case "/bad.zip":
			_, _ = w.Write([]byte("not the expected provider archive"))
		case "/good.zip":
			_, _ = w.Write(archiveBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	meta := func(path string) getproviders.PackageMeta {
		return getproviders.PackageMeta{
			Provider:         provider,
			Version:          version,
			ProtocolVersions: getproviders.VersionList{getproviders.MustParseVersion("5.0.0")},
			TargetPlatform:   platform,
			Filename:         filepath.Base(archive.Name()),
			Location:         getproviders.PackageHTTPURL(server.URL + path),
			Authentication:   getproviders.NewArchiveChecksumAuthentication(platform, checksum),
		}
	}
	newLocks := func() *depsfile.Locks {
		locks := depsfile.NewLocks()
		locks.SetProvider(provider, version, constraints, []getproviders.Hash{packageHash})
		return locks
	}
	globalCachePath := t.TempDir()
	runInstall := func(ctx context.Context, packageMeta getproviders.PackageMeta) error {
		target := NewDirWithPlatform(t.TempDir(), platform)
		installer := NewInstaller(target, getproviders.NewMockSource([]getproviders.PackageMeta{packageMeta}, nil))
		installer.SetGlobalCacheDir(NewDirWithPlatform(globalCachePath, platform))
		_, err := installer.EnsureProviderVersions(
			ctx,
			newLocks(),
			getproviders.Requirements{provider: constraints},
			InstallNewProvidersOnly,
		)
		return err
	}

	if err := runInstall(context.Background(), meta("/bad.zip")); err == nil {
		t.Fatal("checksum-mismatching provider installation succeeded")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := runInstall(ctx, meta("/good.zip")); err != nil {
		t.Fatalf("valid provider installation failed after prior failure: %s", err)
	}
}

func TestEnsureProviderVersionsConcurrent_waitHonorsCancellation(t *testing.T) {
	provider := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/aws")
	version := getproviders.MustParseVersion("5.100.0")
	constraints := getproviders.MustParseVersionConstraints("= 5.100.0")
	platform := getproviders.CurrentPlatform
	globalCachePath := t.TempDir()

	globalCache := NewDirWithPlatform(globalCachePath, platform)
	unlock, err := globalCache.lockProviderVersion(context.Background(), provider, version)
	if err != nil {
		t.Fatalf("failed to acquire provider lock: %s", err)
	}
	defer func() {
		_ = unlock()
	}()

	meta, err := getproviders.FakeInstallablePackageMeta(
		t,
		provider,
		version,
		getproviders.VersionList{getproviders.MustParseVersion("5.0.0")},
		platform,
		"",
	)
	if err != nil {
		t.Fatalf("failed to create provider metadata: %s", err)
	}
	packageHash, err := getproviders.PackageHashV1(meta.Location)
	if err != nil {
		t.Fatalf("failed to calculate provider package hash: %s", err)
	}
	locks := depsfile.NewLocks()
	locks.SetProvider(provider, version, constraints, []getproviders.Hash{packageHash})

	installer := NewInstaller(
		NewDirWithPlatform(t.TempDir(), platform),
		getproviders.NewMockSource([]getproviders.PackageMeta{meta}, nil),
	)
	installer.SetGlobalCacheDir(NewDirWithPlatform(globalCachePath, platform))

	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan error, 1)
	go func() {
		_, err := installer.EnsureProviderVersions(
			ctx,
			locks,
			getproviders.Requirements{provider: constraints},
			InstallNewProvidersOnly,
		)
		resultCh <- err
	}()

	select {
	case err := <-resultCh:
		t.Fatalf("installer returned before cancellation: %v", err)
	case <-time.After(250 * time.Millisecond):
	}
	cancel()

	select {
	case err := <-resultCh:
		var installerErr InstallerError
		if !errors.As(err, &installerErr) {
			t.Fatalf("installer error type = %T; want InstallerError", err)
		}
		if !errors.Is(installerErr.ProviderErrors[provider], context.Canceled) {
			t.Fatalf("provider lock error = %v; want context.Canceled", installerErr.ProviderErrors[provider])
		}
	case <-time.After(5 * time.Second):
		t.Fatal("installer did not return after cancellation")
	}
}
