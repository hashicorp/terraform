// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package depsfile

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	version "github.com/hashicorp/go-version"

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

func TestModuleLocks(t *testing.T) {
	v1_0_0, _ := version.NewVersion("1.0.0")
	hash1 := providerreqs.HashScheme("h1").New("abc123def456")
	hash2 := providerreqs.HashScheme("h1").New("xyz789uvw012")

	t.Run("NewModuleLock", func(t *testing.T) {
		lock := NewModuleLock("vpc", "terraform-aws-modules/vpc/aws", v1_0_0, []providerreqs.Hash{hash1, hash2})

		if got, want := lock.Path(), "vpc"; got != want {
			t.Errorf("wrong path\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := lock.SourceAddr(), "terraform-aws-modules/vpc/aws"; got != want {
			t.Errorf("wrong source address\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := lock.Version(), v1_0_0; got != want {
			t.Errorf("wrong version\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := len(lock.AllHashes()), 2; got != want {
			t.Errorf("wrong number of hashes %d; want %d", got, want)
		}
	})

	t.Run("SetModule and Module", func(t *testing.T) {
		locks := NewLocks()
		modulePath := addrs.Module{"vpc"}

		lock := locks.SetModule(modulePath, "terraform-aws-modules/vpc/aws", v1_0_0, []providerreqs.Hash{hash1})

		retrieved := locks.Module(modulePath)
		if retrieved == nil {
			t.Fatal("expected to retrieve module lock, got nil")
		}
		if retrieved != lock {
			t.Error("retrieved lock is not the same as the one that was set")
		}
	})

	t.Run("AllModules", func(t *testing.T) {
		locks := NewLocks()

		// Start with empty
		modules := locks.AllModules()
		if got, want := len(modules), 0; got != want {
			t.Errorf("wrong number of modules %d; want %d", got, want)
		}

		// Add some modules
		locks.SetModule(addrs.Module{"vpc"}, "source1", v1_0_0, []providerreqs.Hash{hash1})
		locks.SetModule(addrs.Module{"subnet", "private"}, "source2", nil, []providerreqs.Hash{hash2})

		modules = locks.AllModules()
		if got, want := len(modules), 2; got != want {
			t.Errorf("wrong number of modules %d; want %d", got, want)
		}

		// Check that the returned map is a copy
		modules["test"] = &ModuleLock{}
		if got, want := len(locks.AllModules()), 2; got != want {
			t.Errorf("original map was modified; got %d want %d", got, want)
		}
	})

	t.Run("RemoveModule", func(t *testing.T) {
		locks := NewLocks()
		modulePath := addrs.Module{"vpc"}

		locks.SetModule(modulePath, "source", v1_0_0, []providerreqs.Hash{hash1})
		if locks.Module(modulePath) == nil {
			t.Fatal("module should be present before removal")
		}

		locks.RemoveModule(modulePath)
		if locks.Module(modulePath) != nil {
			t.Error("module should be nil after removal")
		}

		// Removing a non-existent module should be a no-op
		locks.RemoveModule(addrs.Module{"nonexistent"})
	})

	t.Run("ModuleKey function", func(t *testing.T) {
		tests := []struct {
			name string
			path addrs.Module
			want string
		}{
			{"root module", addrs.RootModule, ""},
			{"single level", addrs.Module{"vpc"}, "vpc"},
			{"nested", addrs.Module{"vpc", "subnet"}, "vpc.subnet"},
			{"deeply nested", addrs.Module{"a", "b", "c"}, "a.b.c"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				if got := ModuleKey(test.path); got != test.want {
					t.Errorf("ModuleKey(%v) = %q; want %q", test.path, got, test.want)
				}
			})
		}
	})

	t.Run("Locks.Empty with modules", func(t *testing.T) {
		locks := NewLocks()
		if !locks.Empty() {
			t.Error("new locks should be empty")
		}

		// Add a provider
		locks.SetProvider(addrs.NewDefaultProvider("test"), providerreqs.MustParseVersion("1.0.0"), nil, []providerreqs.Hash{hash1})
		if locks.Empty() {
			t.Error("locks with provider should not be empty")
		}

		// Remove provider, add module
		locks.RemoveProvider(addrs.NewDefaultProvider("test"))
		locks.SetModule(addrs.Module{"test"}, "source", nil, []providerreqs.Hash{hash1})
		if locks.Empty() {
			t.Error("locks with module should not be empty")
		}

		// Remove module
		locks.RemoveModule(addrs.Module{"test"})
		if !locks.Empty() {
			t.Error("locks should be empty after removing all")
		}
	})

	t.Run("DeepCopy with modules", func(t *testing.T) {
		original := NewLocks()
		original.SetModule(addrs.Module{"vpc"}, "source", v1_0_0, []providerreqs.Hash{hash1, hash2})
		original.SetModule(addrs.Module{"subnet"}, "source2", nil, []providerreqs.Hash{hash1})

		copy := original.DeepCopy()

		// Verify copy has the same data
		if got, want := len(copy.AllModules()), 2; got != want {
			t.Errorf("wrong number of modules in copy %d; want %d", got, want)
		}

		vpcLock := copy.Module(addrs.Module{"vpc"})
		if vpcLock == nil {
			t.Fatal("vpc module not found in copy")
		}
		if got, want := vpcLock.SourceAddr(), "source"; got != want {
			t.Errorf("wrong source in copy\ngot:  %s\nwant: %s", got, want)
		}

		// Verify it's actually a deep copy by modifying original
		original.RemoveModule(addrs.Module{"vpc"})
		if copy.Module(addrs.Module{"vpc"}) == nil {
			t.Error("copy was affected by modification to original")
		}
	})
}
