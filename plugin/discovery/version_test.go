package discovery

import (
	"reflect"
	"testing"
)

func TestSortVersions(t *testing.T) {
	versions := Versions{
		VersionStr("4").MustParse(),
		VersionStr("3.1").MustParse(),
		VersionStr("1.2").MustParse(),
		VersionStr("1.2.3").MustParse(),
		VersionStr("2.2.3").MustParse(),
		VersionStr("3.2.1").MustParse(),
		VersionStr("2.3.2").MustParse(),
	}

	expected := []string{
		"4.0.0",
		"3.2.1",
		"3.1.0",
		"2.3.2",
		"2.2.3",
		"1.2.3",
		"1.2.0",
	}

	versions.Sort()

	var sorted []string
	for _, v := range versions {
		sorted = append(sorted, v.String())
	}

	if !reflect.DeepEqual(sorted, expected) {
		t.Fatal("versions aren't sorted:", sorted)
	}
}
