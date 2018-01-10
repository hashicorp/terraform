package discovery

import (
	"fmt"
	"testing"
)

func TestPluginConstraintsAllows(t *testing.T) {
	tests := []struct {
		Constraints *PluginConstraints
		Version     string
		Want        bool
	}{
		{
			&PluginConstraints{
				Versions: AllVersions,
			},
			"1.0.0",
			true,
		},
		{
			&PluginConstraints{
				Versions: ConstraintStr(">1.0.0").MustParse(),
			},
			"1.0.0",
			false,
		},
		// This is not an exhaustive test because the callees
		// already have plentiful tests of their own.
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			version := VersionStr(test.Version).MustParse()
			got := test.Constraints.Allows(version)
			if got != test.Want {
				t.Logf("looking for %s in %#v", test.Version, test.Constraints)
				t.Errorf("wrong result %#v; want %#v", got, test.Want)
			}
		})
	}
}

func TestPluginConstraintsAcceptsSHA256(t *testing.T) {
	mustUnhex := func(hex string) (ret []byte) {
		_, err := fmt.Sscanf(hex, "%x", &ret)
		if err != nil {
			panic(err)
		}
		return ret
	}

	tests := []struct {
		Constraints *PluginConstraints
		Digest      []byte
		Want        bool
	}{
		{
			&PluginConstraints{
				Versions: AllVersions,
				SHA256:   mustUnhex("0123456789abcdef"),
			},
			mustUnhex("0123456789abcdef"),
			true,
		},
		{
			&PluginConstraints{
				Versions: AllVersions,
				SHA256:   mustUnhex("0123456789abcdef"),
			},
			mustUnhex("f00dface"),
			false,
		},
		{
			&PluginConstraints{
				Versions: AllVersions,
				SHA256:   nil,
			},
			mustUnhex("f00dface"),
			true,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			got := test.Constraints.AcceptsSHA256(test.Digest)
			if got != test.Want {
				t.Logf("%#v.AcceptsSHA256(%#v)", test.Constraints, test.Digest)
				t.Errorf("wrong result %#v; want %#v", got, test.Want)
			}
		})
	}
}
