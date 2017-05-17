package resource

import (
	"regexp"
	"strings"
	"testing"
)

var allHex = regexp.MustCompile(`^[a-f0-9]+$`)

func TestUniqueId(t *testing.T) {
	iterations := 10000
	ids := make(map[string]struct{})
	var id, lastId string
	for i := 0; i < iterations; i++ {
		id = UniqueId()

		if _, ok := ids[id]; ok {
			t.Fatalf("Got duplicated id! %s", id)
		}

		if !strings.HasPrefix(id, "terraform-") {
			t.Fatalf("Unique ID didn't have terraform- prefix! %s", id)
		}

		rest := strings.TrimPrefix(id, "terraform-")

		if len(rest) != 26 {
			t.Fatalf("Post-prefix part has wrong length! %s", rest)
		}

		if !allHex.MatchString(rest) {
			t.Fatalf("Random part not all hex! %s", rest)
		}

		if lastId != "" && lastId >= id {
			t.Fatalf("IDs not ordered! %s vs %s", lastId, id)
		}

		ids[id] = struct{}{}
		lastId = id
	}
}
