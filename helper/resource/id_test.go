package resource

import (
	"regexp"
	"strings"
	"testing"
)

var allDigits = regexp.MustCompile(`^\d+$`)
var allBase32 = regexp.MustCompile(`^[a-z234567]+$`)

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

		timestamp := rest[:23]
		random := rest[23:]

		if !allDigits.MatchString(timestamp) {
			t.Fatalf("Timestamp not all digits! %s", timestamp)
		}

		if !allBase32.MatchString(random) {
			t.Fatalf("Random part not all base32! %s", random)
		}

		if lastId != "" && lastId >= id {
			t.Fatalf("IDs not ordered! %s vs %s", lastId, id)
		}

		ids[id] = struct{}{}
		lastId = id
	}
}
