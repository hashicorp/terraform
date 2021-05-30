package addrs

import (
	"fmt"
	"testing"
)

func TestResourceEqual_true(t *testing.T) {
	resources := []Resource{
		{
			Mode: ManagedResourceMode,
			Type: "a",
			Name: "b",
		},
		{
			Mode: DataResourceMode,
			Type: "a",
			Name: "b",
		},
	}
	for _, r := range resources {
		t.Run(r.String(), func(t *testing.T) {
			if !r.Equal(r) {
				t.Fatalf("expected %#v to be equal to itself", r)
			}
		})
	}
}

func TestResourceEqual_false(t *testing.T) {
	testCases := []struct {
		left  Resource
		right Resource
	}{
		{
			Resource{Mode: DataResourceMode, Type: "a", Name: "b"},
			Resource{Mode: ManagedResourceMode, Type: "a", Name: "b"},
		},
		{
			Resource{Mode: ManagedResourceMode, Type: "a", Name: "b"},
			Resource{Mode: ManagedResourceMode, Type: "b", Name: "b"},
		},
		{
			Resource{Mode: ManagedResourceMode, Type: "a", Name: "b"},
			Resource{Mode: ManagedResourceMode, Type: "a", Name: "c"},
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s = %s", tc.left, tc.right), func(t *testing.T) {
			if tc.left.Equal(tc.right) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.left, tc.right)
			}

			if tc.right.Equal(tc.left) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.right, tc.left)
			}
		})
	}
}

func TestResourceInstanceEqual_true(t *testing.T) {
	resources := []ResourceInstance{
		{
			Resource: Resource{
				Mode: ManagedResourceMode,
				Type: "a",
				Name: "b",
			},
			Key: IntKey(0),
		},
		{
			Resource: Resource{
				Mode: DataResourceMode,
				Type: "a",
				Name: "b",
			},
			Key: StringKey("x"),
		},
	}
	for _, r := range resources {
		t.Run(r.String(), func(t *testing.T) {
			if !r.Equal(r) {
				t.Fatalf("expected %#v to be equal to itself", r)
			}
		})
	}
}

func TestResourceInstanceEqual_false(t *testing.T) {
	testCases := []struct {
		left  ResourceInstance
		right ResourceInstance
	}{
		{
			ResourceInstance{
				Resource: Resource{Mode: DataResourceMode, Type: "a", Name: "b"},
				Key:      IntKey(0),
			},
			ResourceInstance{
				Resource: Resource{Mode: ManagedResourceMode, Type: "a", Name: "b"},
				Key:      IntKey(0),
			},
		},
		{
			ResourceInstance{
				Resource: Resource{Mode: ManagedResourceMode, Type: "a", Name: "b"},
				Key:      IntKey(0),
			},
			ResourceInstance{
				Resource: Resource{Mode: ManagedResourceMode, Type: "b", Name: "b"},
				Key:      IntKey(0),
			},
		},
		{
			ResourceInstance{
				Resource: Resource{Mode: ManagedResourceMode, Type: "a", Name: "b"},
				Key:      IntKey(0),
			},
			ResourceInstance{
				Resource: Resource{Mode: ManagedResourceMode, Type: "a", Name: "c"},
				Key:      IntKey(0),
			},
		},
		{
			ResourceInstance{
				Resource: Resource{Mode: DataResourceMode, Type: "a", Name: "b"},
				Key:      IntKey(0),
			},
			ResourceInstance{
				Resource: Resource{Mode: DataResourceMode, Type: "a", Name: "b"},
				Key:      StringKey("0"),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s = %s", tc.left, tc.right), func(t *testing.T) {
			if tc.left.Equal(tc.right) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.left, tc.right)
			}

			if tc.right.Equal(tc.left) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.right, tc.left)
			}
		})
	}
}

func TestAbsResourceInstanceEqual_true(t *testing.T) {
	managed := Resource{Mode: ManagedResourceMode, Type: "a", Name: "b"}
	data := Resource{Mode: DataResourceMode, Type: "a", Name: "b"}

	foo, diags := ParseModuleInstanceStr("module.foo")
	if len(diags) > 0 {
		t.Fatalf("unexpected diags: %s", diags.Err())
	}
	foobar, diags := ParseModuleInstanceStr("module.foo[1].module.bar")
	if len(diags) > 0 {
		t.Fatalf("unexpected diags: %s", diags.Err())
	}

	instances := []AbsResourceInstance{
		managed.Instance(IntKey(0)).Absolute(foo),
		data.Instance(IntKey(0)).Absolute(foo),
		managed.Instance(StringKey("a")).Absolute(foobar),
	}
	for _, r := range instances {
		t.Run(r.String(), func(t *testing.T) {
			if !r.Equal(r) {
				t.Fatalf("expected %#v to be equal to itself", r)
			}
		})
	}
}

func TestAbsResourceInstanceEqual_false(t *testing.T) {
	managed := Resource{Mode: ManagedResourceMode, Type: "a", Name: "b"}
	data := Resource{Mode: DataResourceMode, Type: "a", Name: "b"}

	foo, diags := ParseModuleInstanceStr("module.foo")
	if len(diags) > 0 {
		t.Fatalf("unexpected diags: %s", diags.Err())
	}
	foobar, diags := ParseModuleInstanceStr("module.foo[1].module.bar")
	if len(diags) > 0 {
		t.Fatalf("unexpected diags: %s", diags.Err())
	}

	testCases := []struct {
		left  AbsResourceInstance
		right AbsResourceInstance
	}{
		{
			managed.Instance(IntKey(0)).Absolute(foo),
			data.Instance(IntKey(0)).Absolute(foo),
		},
		{
			managed.Instance(IntKey(0)).Absolute(foo),
			managed.Instance(IntKey(0)).Absolute(foobar),
		},
		{
			managed.Instance(IntKey(0)).Absolute(foo),
			managed.Instance(StringKey("0")).Absolute(foo),
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s = %s", tc.left, tc.right), func(t *testing.T) {
			if tc.left.Equal(tc.right) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.left, tc.right)
			}

			if tc.right.Equal(tc.left) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.right, tc.left)
			}
		})
	}
}

func TestConfigResourceEqual_true(t *testing.T) {
	resources := []ConfigResource{
		{
			Resource: Resource{Mode: ManagedResourceMode, Type: "a", Name: "b"},
			Module:   RootModule,
		},
		{
			Resource: Resource{Mode: DataResourceMode, Type: "a", Name: "b"},
			Module:   RootModule,
		},
		{
			Resource: Resource{Mode: ManagedResourceMode, Type: "a", Name: "b"},
			Module:   Module{"foo"},
		},
		{
			Resource: Resource{Mode: DataResourceMode, Type: "a", Name: "b"},
			Module:   Module{"foo"},
		},
	}
	for _, r := range resources {
		t.Run(r.String(), func(t *testing.T) {
			if !r.Equal(r) {
				t.Fatalf("expected %#v to be equal to itself", r)
			}
		})
	}
}

func TestConfigResourceEqual_false(t *testing.T) {
	managed := Resource{Mode: ManagedResourceMode, Type: "a", Name: "b"}
	data := Resource{Mode: DataResourceMode, Type: "a", Name: "b"}

	foo := Module{"foo"}
	foobar := Module{"foobar"}
	testCases := []struct {
		left  ConfigResource
		right ConfigResource
	}{
		{
			ConfigResource{Resource: managed, Module: foo},
			ConfigResource{Resource: data, Module: foo},
		},
		{
			ConfigResource{Resource: managed, Module: foo},
			ConfigResource{Resource: managed, Module: foobar},
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s = %s", tc.left, tc.right), func(t *testing.T) {
			if tc.left.Equal(tc.right) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.left, tc.right)
			}

			if tc.right.Equal(tc.left) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.right, tc.left)
			}
		})
	}
}
