package uuid

import (
	"testing"
)

func TestTimeOrderedUuid(t *testing.T) {
	uuid := TimeOrderedUUID()
	if len(uuid) != 36 {
		t.Fatalf("bad: %s", uuid)
	}
}
