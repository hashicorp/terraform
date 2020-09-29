package getproviders

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/hashicorp/terraform/addrs"
)

// MockSource is an in-memory-only, statically-configured source intended for
// use only in unit tests of other subsystems that consume provider sources.
//
// The MockSource also tracks calls to it in case a calling test wishes to
// assert that particular calls were made.
//
// This should not be used outside of unit test code.
type MockSource struct {
	packages []PackageMeta
	warnings map[addrs.Provider]Warnings
	calls    [][]interface{}
}

var _ Source = (*MockSource)(nil)

// NewMockSource creates and returns a MockSource with the given packages.
//
// The given packages don't necessarily need to refer to objects that actually
// exist on disk or over the network, unless the calling test is planning to
// use (directly or indirectly) the results for further provider installation
// actions.
func NewMockSource(packages []PackageMeta, warns map[addrs.Provider]Warnings) *MockSource {
	return &MockSource{
		packages: packages,
		warnings: warns,
	}
}

// AvailableVersions returns all of the versions of the given provider that
// are available in the fixed set of packages that were passed to
// NewMockSource when creating the receiving source.
func (s *MockSource) AvailableVersions(ctx context.Context, provider addrs.Provider) (VersionList, Warnings, error) {
	s.calls = append(s.calls, []interface{}{"AvailableVersions", provider})
	var ret VersionList
	for _, pkg := range s.packages {
		if pkg.Provider == provider {
			ret = append(ret, pkg.Version)
		}
	}
	var warns []string
	if s.warnings != nil {
		if warnings, ok := s.warnings[provider]; ok {
			warns = warnings
		}
	}
	if len(ret) == 0 {
		// In this case, we'll behave like a registry that doesn't know about
		// this provider at all, rather than just returning an empty result.
		return nil, warns, ErrRegistryProviderNotKnown{provider}
	}
	ret.Sort()
	return ret, warns, nil
}

// PackageMeta returns the first package from the list given to NewMockSource
// when creating the receiver that has the given provider, version, and
// target platform.
//
// If none of the packages match, it returns ErrPlatformNotSupported to
// simulate the situation where a provider release isn't available for a
// particular platform.
//
// Note that if the list of packages passed to NewMockSource contains more
// than one with the same provider, version, and target this function will
// always return the first one in the list, which may not match the behavior
// of other sources in an equivalent situation because it's a degenerate case
// with undefined results.
func (s *MockSource) PackageMeta(ctx context.Context, provider addrs.Provider, version Version, target Platform) (PackageMeta, error) {
	s.calls = append(s.calls, []interface{}{"PackageMeta", provider, version, target})

	for _, pkg := range s.packages {
		if pkg.Provider != provider {
			continue
		}
		if pkg.Version != version {
			// (We're using strict equality rather than precedence here,
			// because this is an exact version specification. The caller
			// should consider precedence when selecting a version in the
			// AvailableVersions response, and pass the exact selected
			// version here.)
			continue
		}
		if pkg.TargetPlatform != target {
			continue
		}
		return pkg, nil
	}

	// If we fall out here then nothing matched at all, so we'll treat that
	// as "platform not supported" for consistency with RegistrySource.
	return PackageMeta{}, ErrPlatformNotSupported{
		Provider: provider,
		Version:  version,
		Platform: target,
	}
}

// CallLog returns a list of calls to other methods of the receiever that have
// been called since it was created, in case a calling test wishes to verify
// a particular sequence of operations.
//
// The result is a slice of slices where the first element of each inner slice
// is the name of the method that was called, and then any subsequent elements
// are positional arguments passed to that method.
//
// Callers are forbidden from modifying any objects accessible via the returned
// value.
func (s *MockSource) CallLog() [][]interface{} {
	return s.calls
}

// FakePackageMeta constructs and returns a PackageMeta that carries the given
// metadata but has fake location information that is likely to fail if
// attempting to install from it.
func FakePackageMeta(provider addrs.Provider, version Version, protocols VersionList, target Platform) PackageMeta {
	return PackageMeta{
		Provider:         provider,
		Version:          version,
		ProtocolVersions: protocols,
		TargetPlatform:   target,

		// Some fake but somewhat-realistic-looking other metadata. This
		// points nowhere, so will fail if attempting to actually use it.
		Filename: fmt.Sprintf("terraform-provider-%s_%s_%s.zip", provider.Type, version.String(), target.String()),
		Location: PackageHTTPURL(fmt.Sprintf("https://fake.invalid/terraform-provider-%s_%s.zip", provider.Type, version.String())),
	}
}

// FakeInstallablePackageMeta constructs and returns a PackageMeta that points
// to a temporary archive file that could actually be installed in principle.
//
// Installing it will not produce a working provider though: just a fake file
// posing as an executable. The filename for the executable defaults to the
// standard terraform-provider-NAME_X.Y.Z format, but can be overridden with
// the execFilename argument.
//
// It's the caller's responsibility to call the close callback returned
// alongside the result in order to clean up the temporary file. The caller
// should call the callback even if this function returns an error, because
// some error conditions leave a partially-created file on disk.
func FakeInstallablePackageMeta(provider addrs.Provider, version Version, protocols VersionList, target Platform, execFilename string) (PackageMeta, func(), error) {
	f, err := ioutil.TempFile("", "terraform-getproviders-fake-package-")
	if err != nil {
		return PackageMeta{}, func() {}, err
	}

	// After this point, all of our return paths should include this as the
	// close callback.
	close := func() {
		f.Close()
		os.Remove(f.Name())
	}

	if execFilename == "" {
		execFilename = fmt.Sprintf("terraform-provider-%s_%s", provider.Type, version.String())
		if target.OS == "windows" {
			// For a little more (technically unnecessary) realism...
			execFilename += ".exe"
		}
	}

	zw := zip.NewWriter(f)
	fw, err := zw.Create(execFilename)
	if err != nil {
		return PackageMeta{}, close, fmt.Errorf("failed to add %s to mock zip file: %s", execFilename, err)
	}
	fmt.Fprintf(fw, "This is a fake provider package for %s %s, not a real provider.\n", provider, version)
	err = zw.Close()
	if err != nil {
		return PackageMeta{}, close, fmt.Errorf("failed to close the mock zip file: %s", err)
	}

	// Compute the SHA256 checksum of the generated file, to allow package
	// authentication code to be exercised.
	f.Seek(0, io.SeekStart)
	h := sha256.New()
	io.Copy(h, f)
	checksum := [32]byte{}
	h.Sum(checksum[:0])

	meta := PackageMeta{
		Provider:         provider,
		Version:          version,
		ProtocolVersions: protocols,
		TargetPlatform:   target,

		Location: PackageLocalArchive(f.Name()),

		// This is a fake filename that mimics what a real registry might
		// indicate as a good filename for this package, in case some caller
		// intends to use it to name a local copy of the temporary file.
		// (At the time of writing, no caller actually does that, but who
		// knows what the future holds?)
		Filename: fmt.Sprintf("terraform-provider-%s_%s_%s.zip", provider.Type, version.String(), target.String()),

		Authentication: NewArchiveChecksumAuthentication(target, checksum),
	}
	return meta, close, nil
}

func (s *MockSource) ForDisplay(provider addrs.Provider) string {
	return "mock source"
}
