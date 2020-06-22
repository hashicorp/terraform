package getproviders

import (
	"sync"

	"github.com/hashicorp/terraform/addrs"
)

// MemoizeSource is a Source that wraps another Source and remembers its
// results so that they can be returned more quickly on future calls to the
// same object.
//
// Each MemoizeSource maintains a cache of response it has seen as part of its
// body. All responses are retained for the remaining lifetime of the object.
// Errors from the underlying source are also cached, and so subsequent calls
// with the same arguments will always produce the same errors.
//
// A MemoizeSource can be called concurrently, with incoming requests processed
// sequentially.
type MemoizeSource struct {
	underlying        Source
	availableVersions map[addrs.Provider]memoizeAvailableVersionsRet
	packageMetas      map[memoizePackageMetaCall]memoizePackageMetaRet
	mu                sync.Mutex
}

type memoizeAvailableVersionsRet struct {
	VersionList VersionList
	Warnings    Warnings
	Err         error
}

type memoizePackageMetaCall struct {
	Provider addrs.Provider
	Version  Version
	Target   Platform
}

type memoizePackageMetaRet struct {
	PackageMeta PackageMeta
	Err         error
}

var _ Source = (*MemoizeSource)(nil)

// NewMemoizeSource constructs and returns a new MemoizeSource that wraps
// the given underlying source and memoizes its results.
func NewMemoizeSource(underlying Source) *MemoizeSource {
	return &MemoizeSource{
		underlying:        underlying,
		availableVersions: make(map[addrs.Provider]memoizeAvailableVersionsRet),
		packageMetas:      make(map[memoizePackageMetaCall]memoizePackageMetaRet),
	}
}

// AvailableVersions requests the available versions from the underlying source
// and caches them before returning them, or on subsequent calls returns the
// result directly from the cache.
func (s *MemoizeSource) AvailableVersions(provider addrs.Provider) (VersionList, Warnings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, exists := s.availableVersions[provider]; exists {
		return existing.VersionList, nil, existing.Err
	}

	ret, warnings, err := s.underlying.AvailableVersions(provider)
	s.availableVersions[provider] = memoizeAvailableVersionsRet{
		VersionList: ret,
		Err:         err,
		Warnings:    warnings,
	}
	return ret, warnings, err
}

// PackageMeta requests package metadata from the underlying source and caches
// the result before returning it, or on subsequent calls returns the result
// directly from the cache.
func (s *MemoizeSource) PackageMeta(provider addrs.Provider, version Version, target Platform) (PackageMeta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := memoizePackageMetaCall{
		Provider: provider,
		Version:  version,
		Target:   target,
	}
	if existing, exists := s.packageMetas[key]; exists {
		return existing.PackageMeta, existing.Err
	}

	ret, err := s.underlying.PackageMeta(provider, version, target)
	s.packageMetas[key] = memoizePackageMetaRet{
		PackageMeta: ret,
		Err:         err,
	}
	return ret, err
}

func (s *MemoizeSource) ForDisplay(provider addrs.Provider) string {
	return s.underlying.ForDisplay(provider)
}
