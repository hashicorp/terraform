package terraform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadStateV1ToV2_noPath(t *testing.T) {
	f, err := os.Open(filepath.Join(fixtureDir, "state-upgrade", "v1-to-v2-empty-path.tfstate"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	s, err := ReadState(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	checkStateString(t, s, "<no state>")
}
