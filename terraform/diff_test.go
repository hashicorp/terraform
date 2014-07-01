package terraform

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestDiff_Empty(t *testing.T) {
	diff := new(Diff)
	if !diff.Empty() {
		t.Fatal("should be empty")
	}

	diff.Resources = map[string]*ResourceDiff{
		"nodeA": &ResourceDiff{},
	}

	if !diff.Empty() {
		t.Fatal("should be empty")
	}

	diff.Resources["nodeA"].Attributes = map[string]*ResourceAttrDiff{
		"foo": &ResourceAttrDiff{
			Old: "foo",
			New: "bar",
		},
	}

	if diff.Empty() {
		t.Fatal("should not be empty")
	}

	diff.Resources["nodeA"].Attributes = nil
	diff.Resources["nodeA"].Destroy = true

	if diff.Empty() {
		t.Fatal("should not be empty")
	}
}

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

func TestResourceDiff_Empty(t *testing.T) {
	var rd *ResourceDiff

	if !rd.Empty() {
		t.Fatal("should be empty")
	}

	rd = new(ResourceDiff)

	if !rd.Empty() {
		t.Fatal("should be empty")
	}

	rd = &ResourceDiff{Destroy: true}

	if rd.Empty() {
		t.Fatal("should not be empty")
	}

	rd = &ResourceDiff{
		Attributes: map[string]*ResourceAttrDiff{
			"foo": &ResourceAttrDiff{
				New: "bar",
			},
		},
	}

	if rd.Empty() {
		t.Fatal("should not be empty")
	}
}

func TestResourceDiff_RequiresNew(t *testing.T) {
	rd := &ResourceDiff{
		Attributes: map[string]*ResourceAttrDiff{
			"foo": &ResourceAttrDiff{},
		},
	}

	if rd.RequiresNew() {
		t.Fatal("should not require new")
	}

	rd.Attributes["foo"].RequiresNew = true

	if !rd.RequiresNew() {
		t.Fatal("should require new")
	}
}

func TestResourceDiff_RequiresNew_nil(t *testing.T) {
	var rd *ResourceDiff

	if rd.RequiresNew() {
		t.Fatal("should not require new")
	}
}

func TestReadWriteDiff(t *testing.T) {
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

	buf := new(bytes.Buffer)
	if err := WriteDiff(diff, buf); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual, err := ReadDiff(buf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(actual, diff) {
		t.Fatalf("bad: %#v", actual)
	}
}

const diffStrBasic = `
CREATE: nodeA
  bar:     "foo" => "<computed>"
  foo:     "foo" => "bar"
  longfoo: "foo" => "bar" (forces new resource)
`
