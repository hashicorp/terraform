package terraform

import (
	"reflect"
	"testing"
)

func TestResourceConfig_CheckSet(t *testing.T) {
	cases := []struct {
		Raw      map[string]interface{}
		Computed []string
		Input    []string
		Errs     bool
	}{
		{
			map[string]interface{}{
				"foo": "bar",
			},
			nil,
			[]string{"foo"},
			false,
		},
		{
			map[string]interface{}{
				"foo": "bar",
			},
			nil,
			[]string{"foo", "bar"},
			true,
		},
		{
			map[string]interface{}{
				"foo": "bar",
			},
			[]string{"bar"},
			[]string{"foo", "bar"},
			false,
		},
	}

	for i, tc := range cases {
		rc := &ResourceConfig{
			ComputedKeys: tc.Computed,
			Raw:          tc.Raw,
		}

		errs := rc.CheckSet(tc.Input)
		if tc.Errs != (len(errs) > 0) {
			t.Fatalf("bad: %d", i)
		}
	}
}

func TestResourceConfig_Get(t *testing.T) {
	cases := []struct {
		Raw      map[string]interface{}
		Computed []string
		Input    string
		Output   interface{}
		OutputOk bool
	}{
		{
			map[string]interface{}{
				"foo": "bar",
			},
			nil,
			"foo",
			"bar",
			true,
		},
		{
			map[string]interface{}{},
			nil,
			"foo",
			nil,
			false,
		},
		{
			map[string]interface{}{
				"foo": map[interface{}]interface{}{
					"bar": "baz",
				},
			},
			nil,
			"foo.bar",
			"baz",
			true,
		},
		{
			map[string]interface{}{
				"foo": []string{
					"one",
					"two",
				},
			},
			nil,
			"foo.1",
			"two",
			true,
		},
	}

	for i, tc := range cases {
		rc := &ResourceConfig{
			ComputedKeys: tc.Computed,
			Raw:          tc.Raw,
		}

		actual, ok := rc.Get(tc.Input)
		if tc.OutputOk != ok {
			t.Fatalf("bad ok: %d", i)
		}
		if !reflect.DeepEqual(tc.Output, actual) {
			t.Fatalf("bad %d: %#v", i, actual)
		}
	}
}

func TestResourceConfig_IsSet(t *testing.T) {
	cases := []struct {
		Raw      map[string]interface{}
		Computed []string
		Input    string
		Output   bool
	}{
		{
			map[string]interface{}{
				"foo": "bar",
			},
			nil,
			"foo",
			true,
		},
		{
			map[string]interface{}{},
			nil,
			"foo",
			false,
		},
		{
			map[string]interface{}{},
			[]string{"foo"},
			"foo",
			true,
		},
		{
			map[string]interface{}{
				"foo": map[interface{}]interface{}{
					"bar": "baz",
				},
			},
			nil,
			"foo.bar",
			true,
		},
	}

	for i, tc := range cases {
		rc := &ResourceConfig{
			ComputedKeys: tc.Computed,
			Raw:          tc.Raw,
		}

		actual := rc.IsSet(tc.Input)
		if actual != tc.Output {
			t.Fatalf("fail case: %d", i)
		}
	}
}

func TestResourceConfig_IsSet_nil(t *testing.T) {
	var rc *ResourceConfig

	if rc.IsSet("foo") {
		t.Fatal("bad")
	}
}

func TestResourceProviderFactoryFixed(t *testing.T) {
	p := new(MockResourceProvider)
	var f ResourceProviderFactory = ResourceProviderFactoryFixed(p)
	actual, err := f()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != p {
		t.Fatal("should be identical")
	}
}
