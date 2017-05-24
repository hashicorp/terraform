package moduledeps

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/plugin/discovery"
)

func TestModuleWalkTree(t *testing.T) {
	type walkStep struct {
		Path       []string
		ParentName string
	}

	tests := []struct {
		Root      *Module
		WalkOrder []walkStep
	}{
		{
			&Module{
				Name:     "root",
				Children: nil,
			},
			[]walkStep{
				{
					Path:       []string{"root"},
					ParentName: "",
				},
			},
		},
		{
			&Module{
				Name: "root",
				Children: []*Module{
					{
						Name: "child",
					},
				},
			},
			[]walkStep{
				{
					Path:       []string{"root"},
					ParentName: "",
				},
				{
					Path:       []string{"root", "child"},
					ParentName: "root",
				},
			},
		},
		{
			&Module{
				Name: "root",
				Children: []*Module{
					{
						Name: "child",
						Children: []*Module{
							{
								Name: "grandchild",
							},
						},
					},
				},
			},
			[]walkStep{
				{
					Path:       []string{"root"},
					ParentName: "",
				},
				{
					Path:       []string{"root", "child"},
					ParentName: "root",
				},
				{
					Path:       []string{"root", "child", "grandchild"},
					ParentName: "child",
				},
			},
		},
		{
			&Module{
				Name: "root",
				Children: []*Module{
					{
						Name: "child1",
						Children: []*Module{
							{
								Name: "grandchild1",
							},
						},
					},
					{
						Name: "child2",
						Children: []*Module{
							{
								Name: "grandchild2",
							},
						},
					},
				},
			},
			[]walkStep{
				{
					Path:       []string{"root"},
					ParentName: "",
				},
				{
					Path:       []string{"root", "child1"},
					ParentName: "root",
				},
				{
					Path:       []string{"root", "child1", "grandchild1"},
					ParentName: "child1",
				},
				{
					Path:       []string{"root", "child2"},
					ParentName: "root",
				},
				{
					Path:       []string{"root", "child2", "grandchild2"},
					ParentName: "child2",
				},
			},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			wo := test.WalkOrder
			test.Root.WalkTree(func(path []string, parent *Module, current *Module) error {
				if len(wo) == 0 {
					t.Fatalf("ran out of walk steps while expecting one for %#v", path)
				}
				step := wo[0]
				wo = wo[1:]
				if got, want := path, step.Path; !reflect.DeepEqual(got, want) {
					t.Errorf("wrong path %#v; want %#v", got, want)
				}
				parentName := ""
				if parent != nil {
					parentName = parent.Name
				}
				if got, want := parentName, step.ParentName; got != want {
					t.Errorf("wrong parent name %q; want %q", got, want)
				}

				if got, want := current.Name, path[len(path)-1]; got != want {
					t.Errorf("mismatching current.Name %q and final path element %q", got, want)
				}
				return nil
			})
		})
	}
}

func TestModuleSortChildren(t *testing.T) {
	m := &Module{
		Name: "root",
		Children: []*Module{
			{
				Name: "apple",
			},
			{
				Name: "zebra",
			},
			{
				Name: "xylophone",
			},
			{
				Name: "pig",
			},
		},
	}

	m.SortChildren()

	want := []string{"apple", "pig", "xylophone", "zebra"}
	var got []string
	for _, c := range m.Children {
		got = append(got, c.Name)
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("wrong order %#v; want %#v", want, got)
	}
}

func TestModulePluginRequirements(t *testing.T) {
	m := &Module{
		Name: "root",
		Providers: Providers{
			"foo": ProviderDependency{
				Constraints: discovery.ConstraintStr(">=1.0.0").MustParse(),
			},
			"foo.bar": ProviderDependency{
				Constraints: discovery.ConstraintStr(">=2.0.0").MustParse(),
			},
			"baz": ProviderDependency{
				Constraints: discovery.ConstraintStr(">=3.0.0").MustParse(),
			},
		},
	}

	reqd := m.PluginRequirements()
	if len(reqd) != 2 {
		t.Errorf("wrong number of elements in %#v; want 2", reqd)
	}
	if got, want := reqd["foo"].Versions.String(), ">=1.0.0,>=2.0.0"; got != want {
		t.Errorf("wrong combination of versions for 'foo' %q; want %q", got, want)
	}
	if got, want := reqd["baz"].Versions.String(), ">=3.0.0"; got != want {
		t.Errorf("wrong combination of versions for 'baz' %q; want %q", got, want)
	}
}
