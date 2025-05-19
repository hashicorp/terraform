// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package depsfile

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
)

func TestLocksEqual(t *testing.T) {
	boopProvider := addrs.NewDefaultProvider("boop")
	v2 := providerreqs.MustParseVersion("2.0.0")
	v2LocalBuild := providerreqs.MustParseVersion("2.0.0+awesomecorp.1")
	v2GtConstraints := providerreqs.MustParseVersionConstraints(">= 2.0.0")
	v2EqConstraints := providerreqs.MustParseVersionConstraints("2.0.0")
	hash1 := providerreqs.HashScheme("test").New("1")
	hash2 := providerreqs.HashScheme("test").New("2")
	hash3 := providerreqs.HashScheme("test").New("3")

	equalBothWays := func(t *testing.T, a, b *Locks) {
		t.Helper()
		if !a.Equal(b) {
			t.Errorf("a should be equal to b")
		}
		if !b.Equal(a) {
			t.Errorf("b should be equal to a")
		}
	}
	nonEqualBothWays := func(t *testing.T, a, b *Locks) {
		t.Helper()
		if a.Equal(b) {
			t.Errorf("a should be equal to b")
		}
		if b.Equal(a) {
			t.Errorf("b should be equal to a")
		}
	}

	t.Run("both empty", func(t *testing.T) {
		a := NewLocks()
		b := NewLocks()
		equalBothWays(t, a, b)
	})
	t.Run("an extra provider lock", func(t *testing.T) {
		a := NewLocks()
		b := NewLocks()
		b.SetProvider(boopProvider, v2, v2GtConstraints, nil)
		nonEqualBothWays(t, a, b)
	})
	t.Run("both have boop provider with same version", func(t *testing.T) {
		a := NewLocks()
		b := NewLocks()
		// Note: the constraints are not part of the definition of "Equal", so they can differ
		a.SetProvider(boopProvider, v2, v2GtConstraints, nil)
		b.SetProvider(boopProvider, v2, v2EqConstraints, nil)
		equalBothWays(t, a, b)
	})
	t.Run("both have boop provider with different versions", func(t *testing.T) {
		a := NewLocks()
		b := NewLocks()
		a.SetProvider(boopProvider, v2, v2EqConstraints, nil)
		b.SetProvider(boopProvider, v2LocalBuild, v2EqConstraints, nil)
		nonEqualBothWays(t, a, b)
	})
	t.Run("both have boop provider with same version and same hashes", func(t *testing.T) {
		a := NewLocks()
		b := NewLocks()
		hashes := []providerreqs.Hash{hash1, hash2, hash3}
		a.SetProvider(boopProvider, v2, v2EqConstraints, hashes)
		b.SetProvider(boopProvider, v2, v2EqConstraints, hashes)
		equalBothWays(t, a, b)
	})
	t.Run("both have boop provider with same version but different hashes", func(t *testing.T) {
		a := NewLocks()
		b := NewLocks()
		hashesA := []providerreqs.Hash{hash1, hash2}
		hashesB := []providerreqs.Hash{hash1, hash3}
		a.SetProvider(boopProvider, v2, v2EqConstraints, hashesA)
		b.SetProvider(boopProvider, v2, v2EqConstraints, hashesB)
		nonEqualBothWays(t, a, b)
	})
}

func TestLocksEqualProviderAddress(t *testing.T) {
	boopProvider := addrs.NewDefaultProvider("boop")
	v2 := providerreqs.MustParseVersion("2.0.0")
	v2LocalBuild := providerreqs.MustParseVersion("2.0.0+awesomecorp.1")
	v2GtConstraints := providerreqs.MustParseVersionConstraints(">= 2.0.0")
	v2EqConstraints := providerreqs.MustParseVersionConstraints("2.0.0")
	hash1 := providerreqs.HashScheme("test").New("1")
	hash2 := providerreqs.HashScheme("test").New("2")
	hash3 := providerreqs.HashScheme("test").New("3")

	equalProviderAddressBothWays := func(t *testing.T, a, b *Locks) {
		t.Helper()
		if !a.EqualProviderAddress(b) {
			t.Errorf("a should be equal to b")
		}
		if !b.EqualProviderAddress(a) {
			t.Errorf("b should be equal to a")
		}
	}
	nonEqualProviderAddressBothWays := func(t *testing.T, a, b *Locks) {
		t.Helper()
		if a.EqualProviderAddress(b) {
			t.Errorf("a should be equal to b")
		}
		if b.EqualProviderAddress(a) {
			t.Errorf("b should be equal to a")
		}
	}

	t.Run("both empty", func(t *testing.T) {
		a := NewLocks()
		b := NewLocks()
		equalProviderAddressBothWays(t, a, b)
	})
	t.Run("an extra provider lock", func(t *testing.T) {
		a := NewLocks()
		b := NewLocks()
		b.SetProvider(boopProvider, v2, v2GtConstraints, nil)
		nonEqualProviderAddressBothWays(t, a, b)
	})
	t.Run("both have boop provider with different versions", func(t *testing.T) {
		a := NewLocks()
		b := NewLocks()
		a.SetProvider(boopProvider, v2, v2EqConstraints, nil)
		b.SetProvider(boopProvider, v2LocalBuild, v2EqConstraints, nil)
		equalProviderAddressBothWays(t, a, b)
	})
	t.Run("both have boop provider with same version but different hashes", func(t *testing.T) {
		a := NewLocks()
		b := NewLocks()
		hashesA := []providerreqs.Hash{hash1, hash2}
		hashesB := []providerreqs.Hash{hash1, hash3}
		a.SetProvider(boopProvider, v2, v2EqConstraints, hashesA)
		b.SetProvider(boopProvider, v2, v2EqConstraints, hashesB)
		equalProviderAddressBothWays(t, a, b)
	})
}

func TestLocksProviderSetRemove(t *testing.T) {
	beepProvider := addrs.NewDefaultProvider("beep")
	boopProvider := addrs.NewDefaultProvider("boop")
	v2 := providerreqs.MustParseVersion("2.0.0")
	v2EqConstraints := providerreqs.MustParseVersionConstraints("2.0.0")
	v2GtConstraints := providerreqs.MustParseVersionConstraints(">= 2.0.0")
	hash := providerreqs.HashScheme("test").New("1")

	locks := NewLocks()
	if got, want := len(locks.AllProviders()), 0; got != want {
		t.Fatalf("fresh locks object already has providers")
	}

	locks.SetProvider(boopProvider, v2, v2EqConstraints, []providerreqs.Hash{hash})
	{
		got := locks.AllProviders()
		want := map[addrs.Provider]*ProviderLock{
			boopProvider: {
				addr:               boopProvider,
				version:            v2,
				versionConstraints: v2EqConstraints,
				hashes:             []providerreqs.Hash{hash},
			},
		}
		if diff := cmp.Diff(want, got, ProviderLockComparer); diff != "" {
			t.Fatalf("wrong providers after SetProvider boop\n%s", diff)
		}
	}

	locks.SetProvider(beepProvider, v2, v2GtConstraints, []providerreqs.Hash{hash})
	{
		got := locks.AllProviders()
		want := map[addrs.Provider]*ProviderLock{
			boopProvider: {
				addr:               boopProvider,
				version:            v2,
				versionConstraints: v2EqConstraints,
				hashes:             []providerreqs.Hash{hash},
			},
			beepProvider: {
				addr:               beepProvider,
				version:            v2,
				versionConstraints: v2GtConstraints,
				hashes:             []providerreqs.Hash{hash},
			},
		}
		if diff := cmp.Diff(want, got, ProviderLockComparer); diff != "" {
			t.Fatalf("wrong providers after SetProvider beep\n%s", diff)
		}
	}

	locks.RemoveProvider(boopProvider)
	{
		got := locks.AllProviders()
		want := map[addrs.Provider]*ProviderLock{
			beepProvider: {
				addr:               beepProvider,
				version:            v2,
				versionConstraints: v2GtConstraints,
				hashes:             []providerreqs.Hash{hash},
			},
		}
		if diff := cmp.Diff(want, got, ProviderLockComparer); diff != "" {
			t.Fatalf("wrong providers after RemoveProvider boop\n%s", diff)
		}
	}

	locks.RemoveProvider(beepProvider)
	{
		got := locks.AllProviders()
		want := map[addrs.Provider]*ProviderLock{}
		if diff := cmp.Diff(want, got, ProviderLockComparer); diff != "" {
			t.Fatalf("wrong providers after RemoveProvider beep\n%s", diff)
		}
	}
}

func TestProviderLockContainsAll(t *testing.T) {
	provider := addrs.NewDefaultProvider("provider")
	v2 := providerreqs.MustParseVersion("2.0.0")
	v2EqConstraints := providerreqs.MustParseVersionConstraints("2.0.0")

	t.Run("non-symmetric", func(t *testing.T) {
		target := NewProviderLock(provider, v2, v2EqConstraints, []providerreqs.Hash{
			"9r3i9a9QmASqMnQM",
			"K43RHM2klOoywtyW",
			"swJPXfuCNhJsTM5c",
		})

		original := NewProviderLock(provider, v2, v2EqConstraints, []providerreqs.Hash{
			"9r3i9a9QmASqMnQM",
			"1ZAChGWUMWn4zmIk",
			"K43RHM2klOoywtyW",
			"HWjRvIuWZ1LVatnc",
			"swJPXfuCNhJsTM5c",
			"KwhJK4p/U2dqbKhI",
		})

		if !original.ContainsAll(target) {
			t.Errorf("orginal should contain all hashes in target")
		}
		if target.ContainsAll(original) {
			t.Errorf("target should not contain all hashes in orginal")
		}
	})

	t.Run("symmetric", func(t *testing.T) {
		target := NewProviderLock(provider, v2, v2EqConstraints, []providerreqs.Hash{
			"9r3i9a9QmASqMnQM",
			"K43RHM2klOoywtyW",
			"swJPXfuCNhJsTM5c",
		})

		original := NewProviderLock(provider, v2, v2EqConstraints, []providerreqs.Hash{
			"9r3i9a9QmASqMnQM",
			"K43RHM2klOoywtyW",
			"swJPXfuCNhJsTM5c",
		})

		if !original.ContainsAll(target) {
			t.Errorf("orginal should contain all hashes in target")
		}
		if !target.ContainsAll(original) {
			t.Errorf("target should not contain all hashes in orginal")
		}
	})

	t.Run("edge case - null", func(t *testing.T) {
		original := NewProviderLock(provider, v2, v2EqConstraints, []providerreqs.Hash{
			"9r3i9a9QmASqMnQM",
			"K43RHM2klOoywtyW",
			"swJPXfuCNhJsTM5c",
		})

		if !original.ContainsAll(nil) {
			t.Fatalf("orginal should report true on nil")
		}
	})

	t.Run("edge case - empty", func(t *testing.T) {
		original := NewProviderLock(provider, v2, v2EqConstraints, []providerreqs.Hash{
			"9r3i9a9QmASqMnQM",
			"K43RHM2klOoywtyW",
			"swJPXfuCNhJsTM5c",
		})

		target := NewProviderLock(provider, v2, v2EqConstraints, []providerreqs.Hash{})

		if !original.ContainsAll(target) {
			t.Fatalf("orginal should report true on empty")
		}
	})

	t.Run("edge case - original empty", func(t *testing.T) {
		original := NewProviderLock(provider, v2, v2EqConstraints, []providerreqs.Hash{})

		target := NewProviderLock(provider, v2, v2EqConstraints, []providerreqs.Hash{
			"9r3i9a9QmASqMnQM",
			"K43RHM2klOoywtyW",
			"swJPXfuCNhJsTM5c",
		})

		if original.ContainsAll(target) {
			t.Fatalf("orginal should report false when empty")
		}
	})
}
