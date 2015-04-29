package resource

import (
	"strings"
	"testing"
)

func TestUniqueId(t *testing.T) {
	iterations := 10000
	ids := make(map[string]struct{})
	var id string
	for i := 0; i < iterations; i++ {
		id = UniqueId()

		if _, ok := ids[id]; ok {
			t.Fatalf("Got duplicated id! %s", id)
		}

		if !strings.HasPrefix(id, "terraform-") {
			t.Fatalf("Unique ID didn't have terraform- prefix! %s", id)
		}

		ids[id] = struct{}{}
	}
}
