// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package depsfile

import (
	"fmt"
	"slices"
	"sort"

	"maps"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
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

	// overriddenProviders is a subset of providers which we might be tracking
	// in field providers but whose lock information we're disregarding for
	// this particular run due to some feature that forces Terraform to not
	// use a normally-installed plugin for it. For example, the "provider dev
	// overrides" feature means that we'll be using an arbitrary directory on
	// disk as the package, regardless of what might be selected in "providers".
	//
	// overriddenProviders is an in-memory-only annotation, never stored as
	// part of a lock file and thus not persistent between Terraform runs.
	// The CLI layer is generally the one responsible for populating this,
	// by calling SetProviderOverridden in response to CLI Configuration
	// settings, environment variables, or whatever similar sources.
	overriddenProviders map[addrs.Provider]struct{}

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

// AllProviders returns a map describing all of the provider locks in the
// receiver.
func (l *Locks) AllProviders() map[addrs.Provider]*ProviderLock {
	// We return a copy of our internal map so that future calls to
	// SetProvider won't modify the map we're returning, or vice-versa.
	return maps.Clone(l.providers)
}

// SetProvider creates a new lock or replaces the existing lock for the given
// provider.
//
// SetProvider returns the newly-created provider lock object, which
// invalidates any ProviderLock object previously returned from Provider or
// SetProvider for the given provider address.
//
// The ownership of the backing array for the slice of hashes passes to this
// function, and so the caller must not read or write that backing array after
// calling SetProvider.
//
// Only lockable providers can be passed to this method. If you pass a
// non-lockable provider address then this function will panic. Use
// function ProviderIsLockable to determine whether a particular provider
// should participate in the version locking mechanism.
func (l *Locks) SetProvider(addr addrs.Provider, version providerreqs.Version, constraints providerreqs.VersionConstraints, hashes []providerreqs.Hash) *ProviderLock {
	if !ProviderIsLockable(addr) {
		panic(fmt.Sprintf("Locks.SetProvider with non-lockable provider %s", addr))
	}

	new := NewProviderLock(addr, version, constraints, hashes)
	l.providers[new.addr] = new
	return new
}

// RemoveProvider removes any existing lock file entry for the given provider.
//
// If the given provider did not already have a lock entry, RemoveProvider is
// a no-op.
//
// Only lockable providers can be passed to this method. If you pass a
// non-lockable provider address then this function will panic. Use
// function ProviderIsLockable to determine whether a particular provider
// should participate in the version locking mechanism.
func (l *Locks) RemoveProvider(addr addrs.Provider) {
	if !ProviderIsLockable(addr) {
		panic(fmt.Sprintf("Locks.RemoveProvider with non-lockable provider %s", addr))
	}

	delete(l.providers, addr)
}

// SetProviderOverridden records that this particular Terraform process will
// not pay attention to the recorded lock entry for the given provider, and
// will instead access that provider's functionality in some other special
// way that isn't sensitive to provider version selections or checksums.
//
// This is an in-memory-only annotation which lives only inside a particular
// Locks object, and is never persisted as part of a saved lock file on disk.
// It's valid to still use other methods of the reciever to access
// already-stored lock information and to update lock information for an
// overridden provider, but some callers may need to use ProviderIsOverridden
// to selectively disregard stored lock information for overridden providers,
// depending on what they intended to use the lock information for.
func (l *Locks) SetProviderOverridden(addr addrs.Provider) {
	if l.overriddenProviders == nil {
		l.overriddenProviders = make(map[addrs.Provider]struct{})
	}
	l.overriddenProviders[addr] = struct{}{}
}

// ProviderIsOverridden returns true only if the given provider address was
// previously registered as overridden by calling SetProviderOverridden.
func (l *Locks) ProviderIsOverridden(addr addrs.Provider) bool {
	_, ret := l.overriddenProviders[addr]
	return ret
}

// SetSameOverriddenProviders updates the receiver to mark as overridden all
// of the same providers already marked as overridden in the other given locks.
//
// This allows propagating override information between different lock objects,
// as if calling SetProviderOverridden for each address already overridden
// in the other given locks. If the reciever already has overridden providers,
// SetSameOverriddenProviders will preserve them.
func (l *Locks) SetSameOverriddenProviders(other *Locks) {
	if other == nil {
		return
	}
	for addr := range other.overriddenProviders {
		l.SetProviderOverridden(addr)
	}
}

// NewProviderLock creates a new ProviderLock object that isn't associated
// with any Locks object.
//
// This is here primarily for testing. Most callers should use Locks.SetProvider
// to construct a new provider lock and insert it into a Locks object at the
// same time.
//
// The ownership of the backing array for the slice of hashes passes to this
// function, and so the caller must not read or write that backing array after
// calling NewProviderLock.
//
// Only lockable providers can be passed to this method. If you pass a
// non-lockable provider address then this function will panic. Use
// function ProviderIsLockable to determine whether a particular provider
// should participate in the version locking mechanism.
func NewProviderLock(addr addrs.Provider, version providerreqs.Version, constraints providerreqs.VersionConstraints, hashes []providerreqs.Hash) *ProviderLock {
	if !ProviderIsLockable(addr) {
		panic(fmt.Sprintf("Locks.NewProviderLock with non-lockable provider %s", addr))
	}

	// Normalize the hashes into lexical order so that we can do straightforward
	// equality tests between different locks for the same provider. The
	// hashes are logically a set, so the given order is insignificant.
	sort.Slice(hashes, func(i, j int) bool {
		return string(hashes[i]) < string(hashes[j])
	})

	// This is a slightly-tricky in-place deduping to avoid unnecessarily
	// allocating a new array in the common case where there are no duplicates:
	// we iterate over "hashes" at the same time as appending to another slice
	// with the same backing array, relying on the fact that deduping can only
	// _skip_ elements from the input, and will never generate additional ones
	// that would cause the writer to get ahead of the reader. This also
	// assumes that we already sorted the items, which means that any duplicates
	// will be consecutive in the sequence.
	dedupeHashes := hashes[:0]
	prevHash := providerreqs.NilHash
	for _, hash := range hashes {
		if hash != prevHash {
			dedupeHashes = append(dedupeHashes, hash)
			prevHash = hash
		}
	}

	return &ProviderLock{
		addr:               addr,
		version:            version,
		versionConstraints: constraints,
		hashes:             dedupeHashes,
	}
}

// ProviderIsLockable returns true if the given provider is eligible for
// version locking.
//
// Currently, all providers except builtin and legacy providers are eligible
// for locking.
func ProviderIsLockable(addr addrs.Provider) bool {
	return !(addr.IsBuiltIn() || addr.IsLegacy())
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

// Equal returns true if the given Locks represents the same information as
// the receiver.
//
// Equal explicitly _does not_ consider the equality of version constraints
// in the saved locks, because those are saved only as hints to help the UI
// explain what's changed between runs, and are never used as part of
// dependency installation decisions.
func (l *Locks) Equal(other *Locks) bool {
	if len(l.providers) != len(other.providers) {
		return false
	}
	for addr, thisLock := range l.providers {
		otherLock, ok := other.providers[addr]
		if !ok {
			return false
		}

		if thisLock.addr != otherLock.addr {
			// It'd be weird to get here because we already looked these up
			// by address above.
			return false
		}
		if thisLock.version != otherLock.version {
			// Equality rather than "Version.Same" because changes to the
			// build metadata are significant for the purpose of this function:
			// it's a different package even if it has the same precedence.
			return false
		}

		// Although "hashes" is declared as a slice, it's logically an
		// unordered set. However, we normalize the slice of hashes when
		// recieving it in NewProviderLock, so we can just do a simple
		// item-by-item equality test here.
		if len(thisLock.hashes) != len(otherLock.hashes) {
			return false
		}
		for i := range thisLock.hashes {
			if thisLock.hashes[i] != otherLock.hashes[i] {
				return false
			}
		}
	}
	// We don't need to worry about providers that are in "other" but not
	// in the receiver, because we tested the lengths being equal above.

	return true
}

// EqualProviderAddress returns true if the given Locks have the same provider
// address as the receiver. This doesn't check version and hashes.
func (l *Locks) EqualProviderAddress(other *Locks) bool {
	if len(l.providers) != len(other.providers) {
		return false
	}

	for addr := range l.providers {
		_, ok := other.providers[addr]
		if !ok {
			return false
		}
	}

	return true
}

// Empty returns true if the given Locks object contains no actual locks.
//
// UI code might wish to use this to distinguish a lock file being
// written for the first time from subsequent updates to that lock file.
func (l *Locks) Empty() bool {
	return len(l.providers) == 0
}

// DeepCopy creates a new Locks that represents the same information as the
// receiver but does not share memory for any parts of the structure that.
// are mutable through methods on Locks.
//
// Note that this does _not_ create deep copies of parts of the structure
// that are technically mutable but are immutable by convention, such as the
// array underlying the slice of version constraints. Callers may mutate the
// resulting data structure only via the direct methods of Locks.
func (l *Locks) DeepCopy() *Locks {
	ret := NewLocks()
	for addr, lock := range l.providers {
		hashes := slices.Clone(lock.hashes)
		ret.SetProvider(addr, lock.version, lock.versionConstraints, hashes)
	}
	return ret
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
	version            providerreqs.Version
	versionConstraints providerreqs.VersionConstraints

	// hashes contains zero or more hashes of packages or package contents
	// for the package associated with the selected version across all of
	// the supported platforms.
	//
	// hashes can contain a mixture of hashes in different formats to support
	// changes over time. The new-style hash format is to have a string
	// starting with "h" followed by a version number and then a colon, like
	// "h1:" for the first hash format version. Other hash versions following
	// this scheme may come later. These versioned hash schemes are implemented
	// in the getproviders package; for example, "h1:" is implemented in
	// providerreqs.HashV1 .
	//
	// There is also a legacy hash format which is just a lowercase-hex-encoded
	// SHA256 hash of the official upstream .zip file for the selected version.
	// We'll allow as that a stop-gap until we can upgrade Terraform Registry
	// to support the new scheme, but is non-ideal because we can verify it only
	// when we have the original .zip file exactly; we can't verify a local
	// directory containing the unpacked contents of that .zip file.
	//
	// We ideally want to populate hashes for all available platforms at
	// once, by referring to the signed checksums file in the upstream
	// registry. In that ideal case it's possible to later work with the same
	// configuration on a different platform while still verifying the hashes.
	// However, installation from any method other than an origin registry
	// means we can only populate the hash for the current platform, and so
	// it won't be possible to verify a subsequent installation of the same
	// provider on a different platform.
	hashes []providerreqs.Hash
}

// Provider returns the address of the provider this lock applies to.
func (l *ProviderLock) Provider() addrs.Provider {
	return l.addr
}

// Version returns the currently-selected version for the corresponding provider.
func (l *ProviderLock) Version() providerreqs.Version {
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
func (l *ProviderLock) VersionConstraints() providerreqs.VersionConstraints {
	return l.versionConstraints
}

// AllHashes returns all of the package hashes that were recorded when this
// lock was created. If no hashes were recorded for that platform, the result
// is a zero-length slice.
//
// If your intent is to verify a package against the recorded hashes, use
// PreferredHashes to get only the hashes which the current version
// of Terraform considers the strongest of the available hashing schemes, one
// of which must match in order for verification to be considered successful.
//
// Do not modify the backing array of the returned slice.
func (l *ProviderLock) AllHashes() []providerreqs.Hash {
	return l.hashes
}

// ContainsAll returns true if the hashes in this ProviderLock contains
// all the hashes in the target.
//
// This function assumes the hashes are in each ProviderLock are sorted.
// If the ProviderLock was created by the NewProviderLock constructor then
// the hashes are guaranteed to be sorted.
func (l *ProviderLock) ContainsAll(target *ProviderLock) bool {
	if target == nil || len(target.hashes) == 0 {
		return true
	}

	targetIndex := 0
	for ix := 0; ix < len(l.hashes); ix++ {
		if l.hashes[ix] == target.hashes[targetIndex] {
			targetIndex++

			if targetIndex >= len(target.hashes) {
				return true
			}
		}
	}
	return false
}

// PreferredHashes returns a filtered version of the AllHashes return value
// which includes only the strongest of the availabile hash schemes, in
// case legacy hash schemes are deprecated over time but still supported for
// upgrade purposes.
//
// At least one of the given hashes must match for a package to be considered
// valud.
func (l *ProviderLock) PreferredHashes() []providerreqs.Hash {
	return providerreqs.PreferredHashes(l.hashes)
}
