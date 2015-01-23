package terraform

import (
	"reflect"
	"strings"
	"testing"
)

func TestDiffEmpty(t *testing.T) {
	diff := new(Diff)
	if !diff.Empty() {
		t.Fatal("should be empty")
	}

	mod := diff.AddModule(rootModulePath)
	mod.Resources["nodeA"] = &InstanceDiff{
		Attributes: map[string]*ResourceAttrDiff{
			"foo": &ResourceAttrDiff{
				Old: "foo",
				New: "bar",
			},
		},
	}

	if diff.Empty() {
		t.Fatal("should not be empty")
	}
}

func TestModuleDiff_ChangeType(t *testing.T) {
	cases := []struct {
		Diff   *ModuleDiff
		Result DiffChangeType
	}{
		{
			&ModuleDiff{},
			DiffNone,
		},
		{
			&ModuleDiff{
				Resources: map[string]*InstanceDiff{
					"foo": &InstanceDiff{Destroy: true},
				},
			},
			DiffDestroy,
		},
		{
			&ModuleDiff{
				Resources: map[string]*InstanceDiff{
					"foo": &InstanceDiff{
						Attributes: map[string]*ResourceAttrDiff{
							"foo": &ResourceAttrDiff{
								Old: "",
								New: "bar",
							},
						},
					},
				},
			},
			DiffUpdate,
		},
		{
			&ModuleDiff{
				Resources: map[string]*InstanceDiff{
					"foo": &InstanceDiff{
						Attributes: map[string]*ResourceAttrDiff{
							"foo": &ResourceAttrDiff{
								Old:         "",
								New:         "bar",
								RequiresNew: true,
							},
						},
					},
				},
			},
			DiffCreate,
		},
		{
			&ModuleDiff{
				Resources: map[string]*InstanceDiff{
					"foo": &InstanceDiff{
						Destroy: true,
						Attributes: map[string]*ResourceAttrDiff{
							"foo": &ResourceAttrDiff{
								Old:         "",
								New:         "bar",
								RequiresNew: true,
							},
						},
					},
				},
			},
			DiffUpdate,
		},
	}

	for i, tc := range cases {
		actual := tc.Diff.ChangeType()
		if actual != tc.Result {
			t.Fatalf("%d: %#v", i, actual)
		}
	}
}

func TestModuleDiff_Empty(t *testing.T) {
	diff := new(ModuleDiff)
	if !diff.Empty() {
		t.Fatal("should be empty")
	}

	diff.Resources = map[string]*InstanceDiff{
		"nodeA": &InstanceDiff{},
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

func TestModuleDiff_String(t *testing.T) {
	diff := &ModuleDiff{
		Resources: map[string]*InstanceDiff{
			"nodeA": &InstanceDiff{
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
	expected := strings.TrimSpace(moduleDiffStrBasic)
	if actual != expected {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestInstanceDiff_ChangeType(t *testing.T) {
	cases := []struct {
		Diff   *InstanceDiff
		Result DiffChangeType
	}{
		{
			&InstanceDiff{},
			DiffNone,
		},
		{
			&InstanceDiff{Destroy: true},
			DiffDestroy,
		},
		{
			&InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"foo": &ResourceAttrDiff{
						Old: "",
						New: "bar",
					},
				},
			},
			DiffUpdate,
		},
		{
			&InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"foo": &ResourceAttrDiff{
						Old:         "",
						New:         "bar",
						RequiresNew: true,
					},
				},
			},
			DiffCreate,
		},
		{
			&InstanceDiff{
				Destroy: true,
				Attributes: map[string]*ResourceAttrDiff{
					"foo": &ResourceAttrDiff{
						Old:         "",
						New:         "bar",
						RequiresNew: true,
					},
				},
			},
			DiffDestroyCreate,
		},
		{
			&InstanceDiff{
				DestroyTainted: true,
				Attributes: map[string]*ResourceAttrDiff{
					"foo": &ResourceAttrDiff{
						Old:         "",
						New:         "bar",
						RequiresNew: true,
					},
				},
			},
			DiffDestroyCreate,
		},
	}

	for i, tc := range cases {
		actual := tc.Diff.ChangeType()
		if actual != tc.Result {
			t.Fatalf("%d: %#v", i, actual)
		}
	}
}

func TestInstanceDiff_Empty(t *testing.T) {
	var rd *InstanceDiff

	if !rd.Empty() {
		t.Fatal("should be empty")
	}

	rd = new(InstanceDiff)

	if !rd.Empty() {
		t.Fatal("should be empty")
	}

	rd = &InstanceDiff{Destroy: true}

	if rd.Empty() {
		t.Fatal("should not be empty")
	}

	rd = &InstanceDiff{
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

func TestModuleDiff_Instances(t *testing.T) {
	yesDiff := &InstanceDiff{Destroy: true}
	noDiff := &InstanceDiff{Destroy: true, DestroyTainted: true}

	cases := []struct {
		Diff   *ModuleDiff
		Id     string
		Result []*InstanceDiff
	}{
		{
			&ModuleDiff{
				Resources: map[string]*InstanceDiff{
					"foo": yesDiff,
					"bar": noDiff,
				},
			},
			"foo",
			[]*InstanceDiff{
				yesDiff,
			},
		},

		{
			&ModuleDiff{
				Resources: map[string]*InstanceDiff{
					"foo":   yesDiff,
					"foo.0": yesDiff,
					"bar":   noDiff,
				},
			},
			"foo",
			[]*InstanceDiff{
				yesDiff,
				yesDiff,
			},
		},

		{
			&ModuleDiff{
				Resources: map[string]*InstanceDiff{
					"foo":     yesDiff,
					"foo.0":   yesDiff,
					"foo_bar": noDiff,
					"bar":     noDiff,
				},
			},
			"foo",
			[]*InstanceDiff{
				yesDiff,
				yesDiff,
			},
		},
	}

	for i, tc := range cases {
		actual := tc.Diff.Instances(tc.Id)
		if !reflect.DeepEqual(actual, tc.Result) {
			t.Fatalf("%d: %#v", i, actual)
		}
	}
}

func TestInstanceDiff_RequiresNew(t *testing.T) {
	rd := &InstanceDiff{
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

func TestInstanceDiff_RequiresNew_nil(t *testing.T) {
	var rd *InstanceDiff

	if rd.RequiresNew() {
		t.Fatal("should not require new")
	}
}

func TestInstanceDiffSame(t *testing.T) {
	cases := []struct {
		One, Two *InstanceDiff
		Same     bool
	}{
		{
			&InstanceDiff{},
			&InstanceDiff{},
			true,
		},

		{
			nil,
			nil,
			true,
		},

		{
			&InstanceDiff{Destroy: false},
			&InstanceDiff{Destroy: true},
			false,
		},

		{
			&InstanceDiff{Destroy: true},
			&InstanceDiff{Destroy: true},
			true,
		},

		{
			&InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"foo": &ResourceAttrDiff{},
				},
			},
			&InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"foo": &ResourceAttrDiff{},
				},
			},
			true,
		},

		{
			&InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"bar": &ResourceAttrDiff{},
				},
			},
			&InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"foo": &ResourceAttrDiff{},
				},
			},
			false,
		},

		// Extra attributes
		{
			&InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"foo": &ResourceAttrDiff{},
				},
			},
			&InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"foo": &ResourceAttrDiff{},
					"bar": &ResourceAttrDiff{},
				},
			},
			false,
		},

		{
			&InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"foo": &ResourceAttrDiff{RequiresNew: true},
				},
			},
			&InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"foo": &ResourceAttrDiff{RequiresNew: false},
				},
			},
			false,
		},

		{
			&InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"foo.#": &ResourceAttrDiff{NewComputed: true},
				},
			},
			&InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"foo.#": &ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"foo.0": &ResourceAttrDiff{
						Old: "",
						New: "12",
					},
				},
			},
			true,
		},

		{
			&InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"foo.#": &ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"foo.~35964334.bar": &ResourceAttrDiff{
						Old: "",
						New: "${var.foo}",
					},
				},
			},
			&InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"foo.#": &ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"foo.87654323.bar": &ResourceAttrDiff{
						Old: "",
						New: "12",
					},
				},
			},
			true,
		},

		{
			&InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"foo.#": &ResourceAttrDiff{
						Old:         "0",
						NewComputed: true,
					},
				},
			},
			&InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{},
			},
			true,
		},
	}

	for i, tc := range cases {
		actual := tc.One.Same(tc.Two)
		if actual != tc.Same {
			t.Fatalf("%d:\n\n%#v\n\n%#v", i, tc.One, tc.Two)
		}
	}
}

const moduleDiffStrBasic = `
CREATE: nodeA
  bar:     "foo" => "<computed>"
  foo:     "foo" => "bar"
  longfoo: "foo" => "bar" (forces new resource)
`
