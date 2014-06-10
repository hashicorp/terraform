package terraform

import (
	"strings"
	"testing"
)

func TestDiff_String(t *testing.T) {
	diff := &Diff{
		Resources: map[string]*ResourceDiff{
			"nodeA": &ResourceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"foo": &ResourceAttrDiff{
						Old: "foo",
						New: "bar",
					},
					"bar": &ResourceAttrDiff{
						Old:         "foo",
						NewComputed: true,
					},
					"longfoo": &ResourceAttrDiff{
						Old:         "foo",
						New:         "bar",
						RequiresNew: true,
					},
				},
			},
		},
	}

	actual := strings.TrimSpace(diff.String())
	expected := strings.TrimSpace(diffStrBasic)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

const diffStrBasic = `
CREATE: nodeA
  bar:     "foo" => "<computed>"
  foo:     "foo" => "bar"
  longfoo: "foo" => "bar" (forces new resource)
`
