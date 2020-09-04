package depsfile

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
)

// Locks is the top-level type representing the information retained in a
// dependency lock file.
//
// Locks and the other types used within it are mutable via various setter
// methods, but they are not safe for concurrent  modifications, so it's the
// caller's responsibility to prevent concurrent writes and writes concurrent
// with reads.
type Locks struct {
	providers map[addrs.Provider]*ProviderLock

	// TODO: In future we'll also have module locks, but the design of that
	// still needs some more work and we're deferring that to get the
	// provider locking capability out sooner, because it's more common to
	// directly depend on providers maintained outside your organization than
	// modules maintained outside your organization.

	// sources is a copy of the map of source buffers produced by the HCL
	// parser during loading, which we retain only so that the caller can
	// use it to produce source code snippets in error messages.
	sources map[string][]byte
}

// NewLocks constructs and returns a new Locks object that initially contains
// no locks at all.
func NewLocks() *Locks {
	return &Locks{
		providers: make(map[addrs.Provider]*ProviderLock),

		// no "sources" here, because that's only for locks objects loaded
		// from files.
	}
}

// Provider returns the stored lock for the given provider, or nil if that
// provider currently has no lock.
func (l *Locks) Provider(addr addrs.Provider) *ProviderLock {
	return l.providers[addr]
}

// SetProvider creates a new lock or replaces the existing lock for the given
// provider.
//
// SetProvider returns the newly-created provider lock object, which
// invalidates any ProviderLock object previously returned from Provider or
// SetProvider for the given provider address.
func (l *Locks) SetProvider(addr addrs.Provider, version getproviders.Version, constraints getproviders.VersionConstraints, hashes map[getproviders.Platform][]string) *ProviderLock {
	new := &ProviderLock{
		addr:               addr,
		version:            version,
		versionConstraints: constraints,
		hashes:             hashes,
	}
	l.providers[addr] = new
	return new
}

// Sources returns the source code of the file the receiver was generated from,
// or an empty map if the receiver wasn't generated from a file.
//
// This return type matches the one expected by HCL diagnostics printers to
// produce source code snapshots, which is the only intended use for this
// method.
func (l *Locks) Sources() map[string][]byte {
	return l.sources
}

// ProviderLock represents lock information for a specific provider.
type ProviderLock struct {
	// addr is the address of the provider this lock applies to.
	addr addrs.Provider

	// version is the specific version that was previously selected, while
	// versionConstraints is the constraint that was used to make that
	// selection, which we can potentially use to hint to run
	// e.g. terraform init -upgrade if a user has changed a version
	// constraint but the previous selection still remains valid.
	// "version" is therefore authoritative, while "versionConstraints" is
	// just for a UI hint and not used to make any real decisions.
	version            getproviders.Version
	versionConstraints getproviders.VersionConstraints

	// hashes contains one or more hashes of packages or package contents
	// for the package associated with the selected version on each supported
	// architecture.
	//
	// hashes can contain a mixture of hashes in different formats to support
	// changes over time. The new-style hash format is to have a string
	// starting with "h" followed by a version number and then a colon, like
	// "h1:" for the first hash format version. Other hash versions following
	// this scheme may come later. These versioned hash schemes are implemented
	// in the getproviders package; for example, "h1:" is implemented in
	// getproviders.HashV1 .
	//
	// There is also a legacy hash format which is just a lowercase-hex-encoded
	// SHA256 hash of the official upstream .zip file for the selected version.
	// We'll allow as that a stop-gap until we can upgrade Terraform Registry
	// to support the new scheme, but is non-ideal because we can verify it only
	// when we have the original .zip file exactly; we can't verify a local
	// directory containing the unpacked contents of that .zip file.
	//
	// We ideally want to populate hashes for all available architectures at
	// once, by referring to the signed checksums file in the upstream
	// registry. In that ideal case it's possible to later work with the same
	// configuration on a different platform while still verifying the hashes.
	// However, installation from any method other than an origin registry
	// means we can only populate the hash for the current platform, and so
	// it won't be possible to verify a subsequent installation of the same
	// provider on a different platform.
	hashes map[getproviders.Platform][]string
}

// Provider returns the address of the provider this lock applies to.
func (l *ProviderLock) Provider() addrs.Provider {
	return l.addr
}

// Version returns the currently-selected version for the corresponding provider.
func (l *ProviderLock) Version() getproviders.Version {
	return l.version
}

// VersionConstraints returns the version constraints that were recorded as
// being used to choose the version returned by Version.
//
// These version constraints are not authoritative for future selections and
// are included only so Terraform can detect if the constraints in
// configuration have changed since a selection was made, and thus hint to the
// user that they may need to run terraform init -upgrade to apply the new
// constraints.
func (l *ProviderLock) VersionConstraints() getproviders.VersionConstraints {
	return l.versionConstraints
}

// HashesForPlatform returns all of the package hashes that were recorded for
// the given platform when this lock was created. If no hashes were recorded
// for that platform, the result is a zero-length slice.
//
// If your intent is to verify a package against the recorded hashes, use
// PreferredHashForPlatform to get a single hash which the current version
// of Terraform considers the strongest of the available hashes, which is
// the one that must pass for verification to be considered successful.
//
// Do not modify the backing array of the returned slice.
func (l *ProviderLock) HashesForPlatform(platform getproviders.Platform) []string {
	return l.hashes[platform]
}

// PreferredHashForPlatform returns a single hash which must match for a package
// for the given platform to be considered valid, or an empty string if there
// are no acceptable hashes recorded for the given platform.
func (l *ProviderLock) PreferredHashForPlatform(platform getproviders.Platform) string {
	return getproviders.PreferredHash(l.hashes[platform])
}
