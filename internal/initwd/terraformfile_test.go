package initwd

import (
	"testing"
)

func TestNewTerraformfile(t *testing.T) {
	tfile, err := NewTerraformfile()
	if err != nil {
		t.Fatalf("unexpected error from NewLoader: %s", err)
	}

	//expected empty tfile
	entry, ok := tfile.GetTerraformEntryOk("foo/bar/test")
	if ok {
		t.Fatalf("unexpected ok found entry %v", entry)
	}
}
