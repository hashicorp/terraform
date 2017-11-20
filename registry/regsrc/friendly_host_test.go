package regsrc

import (
	"strings"
	"testing"
)

func TestFriendlyHost(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		wantHost    string
		wantDisplay string
		wantNorm    string
		wantValid   bool
	}{
		{
			name:        "simple ascii",
			source:      "registry.terraform.io",
			wantHost:    "registry.terraform.io",
			wantDisplay: "registry.terraform.io",
			wantNorm:    "registry.terraform.io",
			wantValid:   true,
		},
		{
			name:        "mixed-case ascii",
			source:      "Registry.TerraForm.io",
			wantHost:    "Registry.TerraForm.io",
			wantDisplay: "registry.terraform.io", // Display case folded
			wantNorm:    "registry.terraform.io",
			wantValid:   true,
		},
		{
			name:        "IDN",
			source:      "ʎɹʇsıƃǝɹ.ɯɹoɟɐɹɹǝʇ.io",
			wantHost:    "ʎɹʇsıƃǝɹ.ɯɹoɟɐɹɹǝʇ.io",
			wantDisplay: "ʎɹʇsıƃǝɹ.ɯɹoɟɐɹɹǝʇ.io",
			wantNorm:    "xn--s-fka0wmm0zea7g8b.xn--o-8ta85a3b1dwcda1k.io",
			wantValid:   true,
		},
		{
			name:        "IDN TLD",
			source:      "zhongwen.中国",
			wantHost:    "zhongwen.中国",
			wantDisplay: "zhongwen.中国",
			wantNorm:    "zhongwen.xn--fiqs8s",
			wantValid:   true,
		},
		{
			name:        "IDN Case Folding",
			source:      "Испытание.com",
			wantHost:    "Испытание.com", // Raw input retains case
			wantDisplay: "испытание.com", // Display form is unicode but case-folded
			wantNorm:    "xn--80akhbyknj4f.com",
			wantValid:   true,
		},
		{
			name:        "Punycode is invalid as an input format",
			source:      "xn--s-fka0wmm0zea7g8b.xn--o-8ta85a3b1dwcda1k.io",
			wantHost:    "xn--s-fka0wmm0zea7g8b.xn--o-8ta85a3b1dwcda1k.io",
			wantDisplay: "ʎɹʇsıƃǝɹ.ɯɹoɟɐɹɹǝʇ.io",
			wantNorm:    InvalidHostString,
			wantValid:   false,
		},
		{
			name:        "non-host prefix is left alone",
			source:      "foo/bar/baz",
			wantHost:    "",
			wantDisplay: "",
			wantNorm:    "",
			wantValid:   false,
		},
	}
	for _, tt := range tests {
		// Matrix each test with prefix and total match variants
		for _, sfx := range []string{"", "/", "/foo/bar/baz"} {
			t.Run(tt.name+" suffix:"+sfx, func(t *testing.T) {
				gotHost, gotRest := ParseFriendlyHost(tt.source + sfx)

				if gotHost == nil {
					if tt.wantHost != "" {
						t.Fatalf("ParseFriendlyHost() gotHost = nil, want %v", tt.wantHost)
					}
					// If we return nil host, the whole input string should be in rest
					if gotRest != (tt.source + sfx) {
						t.Fatalf("ParseFriendlyHost() was nil rest = %s, want %v", gotRest, tt.source+sfx)
					}
					return
				}

				if tt.wantHost == "" {
					t.Fatalf("ParseFriendlyHost() gotHost.Raw = %v, want nil", gotHost.Raw)
				}

				if v := gotHost.String(); v != tt.wantHost {
					t.Fatalf("String() = %v, want %v", v, tt.wantHost)
				}
				if v := gotHost.Display(); v != tt.wantDisplay {
					t.Fatalf("Display() = %v, want %v", v, tt.wantDisplay)
				}
				if v := gotHost.Normalized(); v != tt.wantNorm {
					t.Fatalf("Normalized() = %v, want %v", v, tt.wantNorm)
				}
				if v := gotHost.Valid(); v != tt.wantValid {
					t.Fatalf("Valid() = %v, want %v", v, tt.wantValid)
				}
				if gotRest != strings.TrimLeft(sfx, "/") {
					t.Fatalf("ParseFriendlyHost() rest = %v, want %v", gotRest, strings.TrimLeft(sfx, "/"))
				}

				// Also verify that host compares equal with all the variants.
				if gotHost.Valid() && !gotHost.Equal(&FriendlyHost{Raw: tt.wantDisplay}) {
					t.Fatalf("Equal() should be true for %s and %s", tt.wantHost, tt.wantDisplay)
				}
			})
		}
	}
}

func TestInvalidHostEquals(t *testing.T) {
	invalid := NewFriendlyHost("NOT_A_HOST_NAME")
	valid := PublicRegistryHost

	// invalid hosts are not comparable
	if invalid.Equal(invalid) {
		t.Fatal("invalid host names are not comparable")
	}

	if valid.Equal(invalid) {
		t.Fatalf("%q is not equal to %q", valid, invalid)
	}

	puny := NewFriendlyHost("xn--s-fka0wmm0zea7g8b.xn--o-8ta85a3b1dwcda1k.io")
	display := NewFriendlyHost("ʎɹʇsıƃǝɹ.ɯɹoɟɐɹɹǝʇ.io")

	// The pre-normalized host is not a valid source, and therefore not
	// comparable to the display version.
	if display.Equal(puny) {
		t.Fatalf("invalid host %q should not be comparable", puny)
	}
}
