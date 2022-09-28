package getproviders

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
)

func TestMultiSourceAvailableVersions(t *testing.T) {
	platform1 := Platform{OS: "amigaos", Arch: "m68k"}
	platform2 := Platform{OS: "aros", Arch: "arm"}

	t.Run("unfiltered merging", func(t *testing.T) {
		s1 := NewMockSource([]PackageMeta{
			FakePackageMeta(
				addrs.NewDefaultProvider("foo"),
				MustParseVersion("1.0.0"),
				VersionList{MustParseVersion("5.0")},
				platform1,
			),
			FakePackageMeta(
				addrs.NewDefaultProvider("foo"),
				MustParseVersion("1.0.0"),
				VersionList{MustParseVersion("5.0")},
				platform2,
			),
			FakePackageMeta(
				addrs.NewDefaultProvider("bar"),
				MustParseVersion("1.0.0"),
				VersionList{MustParseVersion("5.0")},
				platform2,
			),
		},
			nil,
		)
		s2 := NewMockSource([]PackageMeta{
			FakePackageMeta(
				addrs.NewDefaultProvider("foo"),
				MustParseVersion("1.0.0"),
				VersionList{MustParseVersion("5.0")},
				platform1,
			),
			FakePackageMeta(
				addrs.NewDefaultProvider("foo"),
				MustParseVersion("1.2.0"),
				VersionList{MustParseVersion("5.0")},
				platform1,
			),
			FakePackageMeta(
				addrs.NewDefaultProvider("bar"),
				MustParseVersion("1.0.0"),
				VersionList{MustParseVersion("5.0")},
				platform1,
			),
		},
			nil,
		)
		multi := MultiSource{
			{Source: s1},
			{Source: s2},
		}

		// AvailableVersions produces the union of all versions available
		// across all of the sources.
		got, _, err := multi.AvailableVersions(context.Background(), addrs.NewDefaultProvider("foo"))
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		want := VersionList{
			MustParseVersion("1.0.0"),
			MustParseVersion("1.2.0"),
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}

		_, _, err = multi.AvailableVersions(context.Background(), addrs.NewDefaultProvider("baz"))
		if want, ok := err.(ErrRegistryProviderNotKnown); !ok {
			t.Fatalf("wrong error type:\ngot:  %T\nwant: %T", err, want)
		}
	})

	t.Run("merging with filters", func(t *testing.T) {
		// This is just testing that filters are being honored at all, using a
		// specific pair of filters. The different filter combinations
		// themselves are tested in TestMultiSourceSelector.

		s1 := NewMockSource([]PackageMeta{
			FakePackageMeta(
				addrs.NewDefaultProvider("foo"),
				MustParseVersion("1.0.0"),
				VersionList{MustParseVersion("5.0")},
				platform1,
			),
			FakePackageMeta(
				addrs.NewDefaultProvider("bar"),
				MustParseVersion("1.0.0"),
				VersionList{MustParseVersion("5.0")},
				platform1,
			),
		},
			nil,
		)
		s2 := NewMockSource([]PackageMeta{
			FakePackageMeta(
				addrs.NewDefaultProvider("foo"),
				MustParseVersion("1.2.0"),
				VersionList{MustParseVersion("5.0")},
				platform1,
			),
			FakePackageMeta(
				addrs.NewDefaultProvider("bar"),
				MustParseVersion("1.2.0"),
				VersionList{MustParseVersion("5.0")},
				platform1,
			),
		},
			nil,
		)
		multi := MultiSource{
			{
				Source:  s1,
				Include: mustParseMultiSourceMatchingPatterns("hashicorp/*"),
			},
			{
				Source:  s2,
				Include: mustParseMultiSourceMatchingPatterns("hashicorp/bar"),
			},
		}

		got, _, err := multi.AvailableVersions(context.Background(), addrs.NewDefaultProvider("foo"))
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		want := VersionList{
			MustParseVersion("1.0.0"),
			// 1.2.0 isn't present because s3 doesn't include "foo"
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}

		got, _, err = multi.AvailableVersions(context.Background(), addrs.NewDefaultProvider("bar"))
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		want = VersionList{
			MustParseVersion("1.0.0"),
			MustParseVersion("1.2.0"), // included because s2 matches "bar"
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}

		_, _, err = multi.AvailableVersions(context.Background(), addrs.NewDefaultProvider("baz"))
		if want, ok := err.(ErrRegistryProviderNotKnown); !ok {
			t.Fatalf("wrong error type:\ngot:  %T\nwant: %T", err, want)
		}
	})

	t.Run("provider not found", func(t *testing.T) {
		s1 := NewMockSource(nil, nil)
		s2 := NewMockSource(nil, nil)
		multi := MultiSource{
			{Source: s1},
			{Source: s2},
		}

		_, _, err := multi.AvailableVersions(context.Background(), addrs.NewDefaultProvider("foo"))
		if err == nil {
			t.Fatal("expected error, got success")
		}

		wantErr := `provider registry registry.terraform.io does not have a provider named registry.terraform.io/hashicorp/foo`

		if err.Error() != wantErr {
			t.Fatalf("wrong error.\ngot:  %s\nwant: %s\n", err, wantErr)
		}

	})

	t.Run("merging with warnings", func(t *testing.T) {
		platform1 := Platform{OS: "amigaos", Arch: "m68k"}
		platform2 := Platform{OS: "aros", Arch: "arm"}
		s1 := NewMockSource([]PackageMeta{
			FakePackageMeta(
				addrs.NewDefaultProvider("bar"),
				MustParseVersion("1.0.0"),
				VersionList{MustParseVersion("5.0")},
				platform2,
			),
		},
			map[addrs.Provider]Warnings{
				addrs.NewDefaultProvider("bar"): {"WARNING!"},
			},
		)
		s2 := NewMockSource([]PackageMeta{
			FakePackageMeta(
				addrs.NewDefaultProvider("bar"),
				MustParseVersion("1.0.0"),
				VersionList{MustParseVersion("5.0")},
				platform1,
			),
		},
			nil,
		)
		multi := MultiSource{
			{Source: s1},
			{Source: s2},
		}

		// AvailableVersions produces the union of all versions available
		// across all of the sources.
		got, warns, err := multi.AvailableVersions(context.Background(), addrs.NewDefaultProvider("bar"))
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		want := VersionList{
			MustParseVersion("1.0.0"),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}

		if len(warns) != 1 {
			t.Fatalf("wrong number of warnings. Got %d, wanted 1", len(warns))
		}
		if warns[0] != "WARNING!" {
			t.Fatalf("wrong warnings. Got %s, wanted \"WARNING!\"", warns[0])
		}
	})
}

func TestMultiSourcePackageMeta(t *testing.T) {
	platform1 := Platform{OS: "amigaos", Arch: "m68k"}
	platform2 := Platform{OS: "aros", Arch: "arm"}

	// We'll use the Filename field of the fake PackageMetas we created above
	// to create a difference between the packages in s1 and the ones in s2,
	// so we can test where individual packages came from below.
	fakeFilename := func(fn string, meta PackageMeta) PackageMeta {
		meta.Filename = fn
		return meta
	}

	onlyInS1 := fakeFilename("s1", FakePackageMeta(
		addrs.NewDefaultProvider("foo"),
		MustParseVersion("1.0.0"),
		VersionList{MustParseVersion("5.0")},
		platform2,
	))
	onlyInS2 := fakeFilename("s2", FakePackageMeta(
		addrs.NewDefaultProvider("foo"),
		MustParseVersion("1.2.0"),
		VersionList{MustParseVersion("5.0")},
		platform1,
	))
	inBothS1 := fakeFilename("s1", FakePackageMeta(
		addrs.NewDefaultProvider("foo"),
		MustParseVersion("1.0.0"),
		VersionList{MustParseVersion("5.0")},
		platform1,
	))
	inBothS2 := fakeFilename("s2", inBothS1)
	s1 := NewMockSource([]PackageMeta{
		inBothS1,
		onlyInS1,
		fakeFilename("s1", FakePackageMeta(
			addrs.NewDefaultProvider("bar"),
			MustParseVersion("1.0.0"),
			VersionList{MustParseVersion("5.0")},
			platform2,
		)),
	},
		nil,
	)
	s2 := NewMockSource([]PackageMeta{
		inBothS2,
		onlyInS2,
		fakeFilename("s2", FakePackageMeta(
			addrs.NewDefaultProvider("bar"),
			MustParseVersion("1.0.0"),
			VersionList{MustParseVersion("5.0")},
			platform1,
		)),
	}, nil)
	multi := MultiSource{
		{Source: s1},
		{Source: s2},
	}

	t.Run("only in s1", func(t *testing.T) {
		got, err := multi.PackageMeta(
			context.Background(),
			addrs.NewDefaultProvider("foo"),
			MustParseVersion("1.0.0"),
			platform2,
		)
		want := onlyInS1
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("only in s2", func(t *testing.T) {
		got, err := multi.PackageMeta(
			context.Background(),
			addrs.NewDefaultProvider("foo"),
			MustParseVersion("1.2.0"),
			platform1,
		)
		want := onlyInS2
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("in both", func(t *testing.T) {
		got, err := multi.PackageMeta(
			context.Background(),
			addrs.NewDefaultProvider("foo"),
			MustParseVersion("1.0.0"),
			platform1,
		)
		want := inBothS1 // S1 "wins" because it's earlier in the MultiSource
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}

		// Make sure inBothS1 and inBothS2 really are different; if not then
		// that's a test bug which we'd rather catch than have this test
		// accidentally passing without actually checking anything.
		if diff := cmp.Diff(inBothS1, inBothS2); diff == "" {
			t.Fatalf("test bug: inBothS1 and inBothS2 are indistinguishable")
		}
	})
	t.Run("in neither", func(t *testing.T) {
		_, err := multi.PackageMeta(
			context.Background(),
			addrs.NewDefaultProvider("nonexist"),
			MustParseVersion("1.0.0"),
			platform1,
		)
		// This case reports "platform not supported" because it assumes that
		// a caller would only pass to it package versions that were returned
		// by a previousc all to AvailableVersions, and therefore a missing
		// object ought to be valid provider/version but an unsupported
		// platform.
		if want, ok := err.(ErrPlatformNotSupported); !ok {
			t.Fatalf("wrong error type:\ngot:  %T\nwant: %T", err, want)
		}
	})
}

func TestMultiSourceSelector(t *testing.T) {
	emptySource := NewMockSource(nil, nil)

	tests := map[string]struct {
		Selector  MultiSourceSelector
		Provider  addrs.Provider
		WantMatch bool
	}{
		"default provider with no constraints": {
			MultiSourceSelector{
				Source: emptySource,
			},
			addrs.NewDefaultProvider("foo"),
			true,
		},
		"built-in provider with no constraints": {
			MultiSourceSelector{
				Source: emptySource,
			},
			addrs.NewBuiltInProvider("bar"),
			true,
		},

		// Include constraints
		"default provider with include constraint that matches it exactly": {
			MultiSourceSelector{
				Source:  emptySource,
				Include: mustParseMultiSourceMatchingPatterns("hashicorp/foo"),
			},
			addrs.NewDefaultProvider("foo"),
			true,
		},
		"default provider with include constraint that matches it via type wildcard": {
			MultiSourceSelector{
				Source:  emptySource,
				Include: mustParseMultiSourceMatchingPatterns("hashicorp/*"),
			},
			addrs.NewDefaultProvider("foo"),
			true,
		},
		"default provider with include constraint that matches it via namespace wildcard": {
			MultiSourceSelector{
				Source:  emptySource,
				Include: mustParseMultiSourceMatchingPatterns("*/*"),
			},
			addrs.NewDefaultProvider("foo"),
			true,
		},
		"default provider with non-normalized include constraint that matches it via type wildcard": {
			MultiSourceSelector{
				Source:  emptySource,
				Include: mustParseMultiSourceMatchingPatterns("HashiCorp/*"),
			},
			addrs.NewDefaultProvider("foo"),
			true,
		},
		"built-in provider with exact include constraint that does not match it": {
			MultiSourceSelector{
				Source:  emptySource,
				Include: mustParseMultiSourceMatchingPatterns("hashicorp/foo"),
			},
			addrs.NewBuiltInProvider("bar"),
			false,
		},
		"built-in provider with type-wild include constraint that does not match it": {
			MultiSourceSelector{
				Source:  emptySource,
				Include: mustParseMultiSourceMatchingPatterns("hashicorp/*"),
			},
			addrs.NewBuiltInProvider("bar"),
			false,
		},
		"built-in provider with namespace-wild include constraint that does not match it": {
			MultiSourceSelector{
				Source:  emptySource,
				Include: mustParseMultiSourceMatchingPatterns("*/*"),
			},
			// Doesn't match because builtin providers are in "terraform.io",
			// but a pattern with no hostname is for registry.terraform.io.
			addrs.NewBuiltInProvider("bar"),
			false,
		},
		"built-in provider with include constraint that matches it via type wildcard": {
			MultiSourceSelector{
				Source:  emptySource,
				Include: mustParseMultiSourceMatchingPatterns("terraform.io/builtin/*"),
			},
			addrs.NewBuiltInProvider("bar"),
			true,
		},

		// Exclude constraints
		"default provider with exclude constraint that matches it exactly": {
			MultiSourceSelector{
				Source:  emptySource,
				Exclude: mustParseMultiSourceMatchingPatterns("hashicorp/foo"),
			},
			addrs.NewDefaultProvider("foo"),
			false,
		},
		"default provider with exclude constraint that matches it via type wildcard": {
			MultiSourceSelector{
				Source:  emptySource,
				Exclude: mustParseMultiSourceMatchingPatterns("hashicorp/*"),
			},
			addrs.NewDefaultProvider("foo"),
			false,
		},
		"default provider with exact exclude constraint that doesn't match it": {
			MultiSourceSelector{
				Source:  emptySource,
				Exclude: mustParseMultiSourceMatchingPatterns("hashicorp/bar"),
			},
			addrs.NewDefaultProvider("foo"),
			true,
		},
		"default provider with non-normalized exclude constraint that matches it via type wildcard": {
			MultiSourceSelector{
				Source:  emptySource,
				Exclude: mustParseMultiSourceMatchingPatterns("HashiCorp/*"),
			},
			addrs.NewDefaultProvider("foo"),
			false,
		},

		// Both include and exclude in a single selector
		"default provider with exclude wildcard overriding include exact": {
			MultiSourceSelector{
				Source:  emptySource,
				Include: mustParseMultiSourceMatchingPatterns("hashicorp/foo"),
				Exclude: mustParseMultiSourceMatchingPatterns("hashicorp/*"),
			},
			addrs.NewDefaultProvider("foo"),
			false,
		},
		"default provider with exclude wildcard overriding irrelevant include exact": {
			MultiSourceSelector{
				Source:  emptySource,
				Include: mustParseMultiSourceMatchingPatterns("hashicorp/bar"),
				Exclude: mustParseMultiSourceMatchingPatterns("hashicorp/*"),
			},
			addrs.NewDefaultProvider("foo"),
			false,
		},
		"default provider with exclude exact overriding include wildcard": {
			MultiSourceSelector{
				Source:  emptySource,
				Include: mustParseMultiSourceMatchingPatterns("hashicorp/*"),
				Exclude: mustParseMultiSourceMatchingPatterns("hashicorp/foo"),
			},
			addrs.NewDefaultProvider("foo"),
			false,
		},
		"default provider with irrelevant exclude exact overriding include wildcard": {
			MultiSourceSelector{
				Source:  emptySource,
				Include: mustParseMultiSourceMatchingPatterns("hashicorp/*"),
				Exclude: mustParseMultiSourceMatchingPatterns("hashicorp/bar"),
			},
			addrs.NewDefaultProvider("foo"),
			true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Logf("include:  %s", test.Selector.Include)
			t.Logf("exclude:  %s", test.Selector.Exclude)
			t.Logf("provider: %s", test.Provider)
			got := test.Selector.CanHandleProvider(test.Provider)
			want := test.WantMatch
			if got != want {
				t.Errorf("wrong result %t; want %t", got, want)
			}
		})
	}
}

func mustParseMultiSourceMatchingPatterns(strs ...string) MultiSourceMatchingPatterns {
	ret, err := ParseMultiSourceMatchingPatterns(strs)
	if err != nil {
		panic(err)
	}
	return ret
}
