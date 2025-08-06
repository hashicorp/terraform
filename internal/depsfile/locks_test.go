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

func TestModuleLocks(t *testing.T) {
	moduleCall1 := addrs.ModuleCall{Name: "example"}.Absolute(addrs.RootModuleInstance)
	moduleCall2 := addrs.ModuleCall{Name: "other"}.Absolute(addrs.RootModuleInstance)
	hash1 := providerreqs.HashScheme("h1").New("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	hash2 := providerreqs.HashScheme("h1").New("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	hash3 := providerreqs.HashScheme("h1").New("ccccccccccccccccccccccccccccccccccccccccccc")

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
			t.Errorf("a should not be equal to b")
		}
		if b.Equal(a) {
			t.Errorf("b should not be equal to a")
		}
	}

	t.Run("both empty", func(t *testing.T) {
		a := NewLocks()
		b := NewLocks()
		equalBothWays(t, a, b)
	})

	t.Run("an extra module lock", func(t *testing.T) {
		a := NewLocks()
		b := NewLocks()
		b.SetModule(moduleCall1, "./modules/example", "", []providerreqs.Hash{hash1})
		nonEqualBothWays(t, a, b)
	})

	t.Run("both have same module with same hashes", func(t *testing.T) {
		a := NewLocks()
		b := NewLocks()
		hashes := []providerreqs.Hash{hash1, hash2, hash3}
		a.SetModule(moduleCall1, "./modules/example", "", hashes)
		b.SetModule(moduleCall1, "./modules/example", "", hashes)
		equalBothWays(t, a, b)
	})

	t.Run("both have same module with different hashes", func(t *testing.T) {
		a := NewLocks()
		b := NewLocks()
		hashesA := []providerreqs.Hash{hash1, hash2}
		hashesB := []providerreqs.Hash{hash1, hash3}
		a.SetModule(moduleCall1, "./modules/example", "", hashesA)
		b.SetModule(moduleCall1, "./modules/example", "", hashesB)
		nonEqualBothWays(t, a, b)
	})

	t.Run("both have same module with different sources", func(t *testing.T) {
		a := NewLocks()
		b := NewLocks()
		hashes := []providerreqs.Hash{hash1}
		a.SetModule(moduleCall1, "./modules/example", "", hashes)
		b.SetModule(moduleCall1, "./modules/other", "", hashes)
		nonEqualBothWays(t, a, b)
	})

	t.Run("different modules", func(t *testing.T) {
		a := NewLocks()
		b := NewLocks()
		hashes := []providerreqs.Hash{hash1}
		a.SetModule(moduleCall1, "./modules/example", "", hashes)
		b.SetModule(moduleCall2, "./modules/example", "", hashes)
		nonEqualBothWays(t, a, b)
	})
}

func TestModuleLockSetRemove(t *testing.T) {
	moduleCall1 := addrs.ModuleCall{Name: "example"}.Absolute(addrs.RootModuleInstance)
	moduleCall2 := addrs.ModuleCall{Name: "other"}.Absolute(addrs.RootModuleInstance)
	hash := providerreqs.HashScheme("h1").New("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	locks := NewLocks()
	if got, want := len(locks.AllModules()), 0; got != want {
		t.Fatalf("fresh locks object already has modules")
	}

	locks.SetModule(moduleCall1, "./modules/example", "", []providerreqs.Hash{hash})
	{
		got := locks.AllModules()
		want := map[string]*ModuleLock{
			moduleCall1.String(): {
				addr:   moduleCall1,
				source: "./modules/example",
				hashes: []providerreqs.Hash{hash},
			},
		}
		if diff := cmp.Diff(want, got, ModuleLockComparer); diff != "" {
			t.Fatalf("wrong modules after SetModule example\n%s", diff)
		}
	}

	locks.SetModule(moduleCall2, "./modules/other", "", []providerreqs.Hash{hash})
	{
		got := locks.AllModules()
		want := map[string]*ModuleLock{
			moduleCall1.String(): {
				addr:   moduleCall1,
				source: "./modules/example",
				hashes: []providerreqs.Hash{hash},
			},
			moduleCall2.String(): {
				addr:   moduleCall2,
				source: "./modules/other",
				hashes: []providerreqs.Hash{hash},
			},
		}
		if diff := cmp.Diff(want, got, ModuleLockComparer); diff != "" {
			t.Fatalf("wrong modules after SetModule other\n%s", diff)
		}
	}

	locks.RemoveModule(moduleCall1)
	{
		got := locks.AllModules()
		want := map[string]*ModuleLock{
			moduleCall2.String(): {
				addr:   moduleCall2,
				source: "./modules/other",
				hashes: []providerreqs.Hash{hash},
			},
		}
		if diff := cmp.Diff(want, got, ModuleLockComparer); diff != "" {
			t.Fatalf("wrong modules after RemoveModule example\n%s", diff)
		}
	}

	locks.RemoveModule(moduleCall2)
	{
		got := locks.AllModules()
		want := map[string]*ModuleLock{}
		if diff := cmp.Diff(want, got, ModuleLockComparer); diff != "" {
			t.Fatalf("wrong modules after RemoveModule other\n%s", diff)
		}
	}
}

func TestModuleLockPreferredHashes(t *testing.T) {
	moduleCall := addrs.ModuleCall{Name: "example"}.Absolute(addrs.RootModuleInstance)
	hash1 := providerreqs.HashScheme("h1").New("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	hash2 := providerreqs.HashScheme("h1").New("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	hashes := []providerreqs.Hash{hash1, hash2}

	lock := &ModuleLock{
		addr:   moduleCall,
		source: "./modules/example",
		hashes: hashes,
	}

	got := lock.PreferredHashes()
	if diff := cmp.Diff(hashes, got); diff != "" {
		t.Errorf("wrong preferred hashes\n%s", diff)
	}
}

func TestModuleLockAbsModuleCallPreventsCollisions(t *testing.T) {
	locks := NewLocks()

	// Create module structure where multiple modules call child modules with the same name:
	// - module.database (root calls database)
	// - module.parent1.database (parent1 calls database)
	// - module.parent2.database (parent2 calls database)

	// Root-level database call
	rootDatabaseCall := addrs.ModuleCall{Name: "database"}.Absolute(addrs.RootModuleInstance)

	// Nested database calls
	parent1Instance := addrs.RootModuleInstance.Child("parent1", addrs.NoKey)
	parent2Instance := addrs.RootModuleInstance.Child("parent2", addrs.NoKey)

	parent1DatabaseCall := addrs.ModuleCall{Name: "database"}.Absolute(parent1Instance)
	parent2DatabaseCall := addrs.ModuleCall{Name: "database"}.Absolute(parent2Instance)

	hash1 := providerreqs.HashScheme("h1").New("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	hash2 := providerreqs.HashScheme("h1").New("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	hash3 := providerreqs.HashScheme("h1").New("ccccccccccccccccccccccccccccccccccccccccccc")

	// Set locks for all three database modules with different sources and hashes
	locks.SetModule(rootDatabaseCall, "./modules/shared-db", "1.0.0", []providerreqs.Hash{hash1})
	locks.SetModule(parent1DatabaseCall, "./modules/postgres", "2.0.0", []providerreqs.Hash{hash2})
	locks.SetModule(parent2DatabaseCall, "./modules/mysql", "3.0.0", []providerreqs.Hash{hash3})

	// Verify all three locks exist and are different
	rootLock := locks.Module(rootDatabaseCall)
	parent1Lock := locks.Module(parent1DatabaseCall)
	parent2Lock := locks.Module(parent2DatabaseCall)

	if rootLock == nil {
		t.Fatal("Expected rootLock to exist for module.database")
	}
	if parent1Lock == nil {
		t.Fatal("Expected parent1Lock to exist for module.parent1.database")
	}
	if parent2Lock == nil {
		t.Fatal("Expected parent2Lock to exist for module.parent2.database")
	}

	// Verify they have different sources (proving no collision)
	if got, want := rootLock.Source(), "./modules/shared-db"; got != want {
		t.Errorf("rootLock source: got %s, want %s", got, want)
	}
	if got, want := parent1Lock.Source(), "./modules/postgres"; got != want {
		t.Errorf("parent1Lock source: got %s, want %s", got, want)
	}
	if got, want := parent2Lock.Source(), "./modules/mysql"; got != want {
		t.Errorf("parent2Lock source: got %s, want %s", got, want)
	}

	// Verify they have different versions
	if got, want := rootLock.Version(), "1.0.0"; got != want {
		t.Errorf("rootLock version: got %s, want %s", got, want)
	}
	if got, want := parent1Lock.Version(), "2.0.0"; got != want {
		t.Errorf("parent1Lock version: got %s, want %s", got, want)
	}
	if got, want := parent2Lock.Version(), "3.0.0"; got != want {
		t.Errorf("parent2Lock version: got %s, want %s", got, want)
	}

	// Verify the string representations are different (proving AbsModuleCall uniqueness)
	rootKey := rootDatabaseCall.String()
	parent1Key := parent1DatabaseCall.String()
	parent2Key := parent2DatabaseCall.String()

	expectedRootKey := "module.database"
	expectedParent1Key := "module.parent1.module.database"
	expectedParent2Key := "module.parent2.module.database"

	if got, want := rootKey, expectedRootKey; got != want {
		t.Errorf("rootDatabaseCall string: got %s, want %s", got, want)
	}
	if got, want := parent1Key, expectedParent1Key; got != want {
		t.Errorf("parent1DatabaseCall string: got %s, want %s", got, want)
	}
	if got, want := parent2Key, expectedParent2Key; got != want {
		t.Errorf("parent2DatabaseCall string: got %s, want %s", got, want)
	}

	// Verify all three keys are unique
	keys := []string{rootKey, parent1Key, parent2Key}
	for i, key1 := range keys {
		for j, key2 := range keys {
			if i != j && key1 == key2 {
				t.Errorf("Keys should be unique but found duplicate: %s", key1)
			}
		}
	}

	// Verify that we have exactly 3 module locks total
	allModules := locks.AllModules()
	if got, want := len(allModules), 3; got != want {
		t.Errorf("Total module locks: got %d, want %d", got, want)
	}
}
