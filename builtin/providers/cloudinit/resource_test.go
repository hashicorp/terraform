package cloudinit

import (
	"sort"
	"testing"
)

func TestPartOrdering(t *testing.T) {
	parts := cloudInitParts{
		cloudInitPart{
			Order:   1,
			Content: "bar",
		},
		cloudInitPart{
			Order:   0,
			Content: "foo",
		},
		cloudInitPart{
			Order:   2,
			Content: "baz",
		},
	}

	sort.Sort(parts)

	if parts[0].Order != 0 || parts[1].Order != 1 || parts[2].Order != 2 {
		t.Error("CloudInit part ordering is incorrect: %+v", parts)
	}
}
