package resource

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

var all36 = regexp.MustCompile(`^[a-z0-9]+$`)

func TestUniqueId(t *testing.T) {
	split := func(rest string) (timestamp, increment string) {
		return rest[:18], rest[18:]
	}

	const prefix = "terraform-"

	iterations := 10000
	ids := make(map[string]struct{})
	var id string
	for i := 0; i < iterations; i++ {
		id = UniqueId()

		if _, ok := ids[id]; ok {
			t.Fatalf("Got duplicated id! %s", id)
		}

		if !strings.HasPrefix(id, prefix) {
			t.Fatalf("Unique ID didn't have terraform- prefix! %s", id)
		}

		rest := strings.TrimPrefix(id, prefix)

		if !all36.MatchString(rest) {
			t.Fatalf("Suffix isn't in base 36! %s", rest)
		}

		ids[id] = struct{}{}
	}

	id1 := UniqueId()
	time.Sleep(time.Millisecond)
	id2 := UniqueId()
	timestamp1, _ := split(strings.TrimPrefix(id1, prefix))
	timestamp2, _ := split(strings.TrimPrefix(id2, prefix))

	if timestamp1 == timestamp2 {
		t.Fatalf("Timestamp part should update at least once a millisecond %s %s",
			id1, id2)
	}
}
