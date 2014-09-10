package hcl

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDecode_interface(t *testing.T) {
	cases := []struct {
		File string
		Err  bool
		Out  interface{}
	}{
		{
			"basic.hcl",
			false,
			map[string]interface{}{
				"foo": "bar",
				"bar": "${file(\"bing/bong.txt\")}",
			},
		},
		{
			"empty.hcl",
			false,
			map[string]interface{}{
				"resource": []map[string]interface{}{
					map[string]interface{}{
						"foo": []map[string]interface{}{
							map[string]interface{}{},
						},
					},
				},
			},
		},
		{
			"multiline_bad.hcl",
			false,
			map[string]interface{}{"foo": "bar\nbaz\n"},
		},
		{
			"multiline.json",
			false,
			map[string]interface{}{"foo": "bar\nbaz"},
		},
		{
			"scientific.json",
			false,
			map[string]interface{}{
				"a": 1e-10,
				"b": 1e+10,
				"c": 1e10,
			},
		},
		{
			"scientific.hcl",
			false,
			map[string]interface{}{
				"a": 1e-10,
				"b": 1e+10,
				"c": 1e10,
			},
		},
		{
			"terraform_heroku.hcl",
			false,
			map[string]interface{}{
				"name": "terraform-test-app",
				"config_vars": []map[string]interface{}{
					map[string]interface{}{
						"FOO": "bar",
					},
				},
			},
		},
		{
			"structure_multi.hcl",
			false,
			map[string]interface{}{
				"foo": []map[string]interface{}{
					map[string]interface{}{
						"baz": []map[string]interface{}{
							map[string]interface{}{"key": 7},
						},
					},
					map[string]interface{}{
						"bar": []map[string]interface{}{
							map[string]interface{}{"key": 12},
						},
					},
				},
			},
		},
		{
			"structure_multi.json",
			false,
			map[string]interface{}{
				"foo": []map[string]interface{}{
					map[string]interface{}{
						"baz": []map[string]interface{}{
							map[string]interface{}{"key": 7},
						},
						"bar": []map[string]interface{}{
							map[string]interface{}{"key": 12},
						},
					},
				},
			},
		},
		{
			"structure_list.hcl",
			false,
			map[string]interface{}{
				"foo": []map[string]interface{}{
					map[string]interface{}{
						"key": 7,
					},
					map[string]interface{}{
						"key": 12,
					},
				},
			},
		},
		{
			"structure_list.json",
			false,
			map[string]interface{}{
				"foo": []interface{}{
					map[string]interface{}{
						"key": 7,
					},
					map[string]interface{}{
						"key": 12,
					},
				},
			},
		},
		{
			"structure_list_deep.json",
			false,
			map[string]interface{}{
				"bar": []map[string]interface{}{
					map[string]interface{}{
						"foo": []map[string]interface{}{
							map[string]interface{}{
								"name": "terraform_example",
								"ingress": []interface{}{
									map[string]interface{}{
										"from_port": 22,
									},
									map[string]interface{}{
										"from_port": 80,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		d, err := ioutil.ReadFile(filepath.Join(fixtureDir, tc.File))
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		var out interface{}
		err = Decode(&out, string(d))
		if (err != nil) != tc.Err {
			t.Fatalf("Input: %s\n\nError: %s", tc.File, err)
		}

		if !reflect.DeepEqual(out, tc.Out) {
			t.Fatalf("Input: %s\n\n%#v\n\n%#v", tc.File, out, tc.Out)
		}
	}
}

func TestDecode_equal(t *testing.T) {
	cases := []struct {
		One, Two string
	}{
		{
			"basic.hcl",
			"basic.json",
		},
		/*
			{
				"structure.hcl",
				"structure.json",
			},
		*/
		{
			"structure.hcl",
			"structure_flat.json",
		},
		{
			"terraform_heroku.hcl",
			"terraform_heroku.json",
		},
	}

	for _, tc := range cases {
		p1 := filepath.Join(fixtureDir, tc.One)
		p2 := filepath.Join(fixtureDir, tc.Two)

		d1, err := ioutil.ReadFile(p1)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		d2, err := ioutil.ReadFile(p2)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		var i1, i2 interface{}
		err = Decode(&i1, string(d1))
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		err = Decode(&i2, string(d2))
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !reflect.DeepEqual(i1, i2) {
			t.Fatalf(
				"%s != %s\n\n%#v\n\n%#v",
				tc.One, tc.Two,
				i1, i2)
		}
	}
}

func TestDecode_flatMap(t *testing.T) {
	var val map[string]map[string]string

	err := Decode(&val, testReadFile(t, "structure_flatmap.hcl"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := map[string]map[string]string{
		"foo": map[string]string{
			"foo": "bar",
			"key": "7",
		},
	}

	if !reflect.DeepEqual(val, expected) {
		t.Fatalf("Actual: %#v\n\nExpected: %#v", val, expected)
	}
}

func TestDecode_structure(t *testing.T) {
	type V struct {
		Key int
		Foo string
	}

	var actual V

	err := Decode(&actual, testReadFile(t, "flat.hcl"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := V{
		Key: 7,
		Foo: "bar",
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Actual: %#v\n\nExpected: %#v", actual, expected)
	}
}

func TestDecode_structurePtr(t *testing.T) {
	type V struct {
		Key int
		Foo string
	}

	var actual *V

	err := Decode(&actual, testReadFile(t, "flat.hcl"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &V{
		Key: 7,
		Foo: "bar",
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Actual: %#v\n\nExpected: %#v", actual, expected)
	}
}

func TestDecode_structureArray(t *testing.T) {
	// This test is extracted from a failure in Consul (consul.io),
	// hence the interesting structure naming.

	type KeyPolicyType string

	type KeyPolicy struct {
		Prefix string `hcl:",key"`
		Policy KeyPolicyType
	}

	type Policy struct {
		Keys []KeyPolicy `hcl:"key,expand"`
	}

	expected := Policy{
		Keys: []KeyPolicy{
			KeyPolicy{
				Prefix: "",
				Policy: "read",
			},
			KeyPolicy{
				Prefix: "foo/",
				Policy: "write",
			},
			KeyPolicy{
				Prefix: "foo/bar/",
				Policy: "read",
			},
			KeyPolicy{
				Prefix: "foo/bar/baz",
				Policy: "deny",
			},
		},
	}

	files := []string{
		"decode_policy.hcl",
		"decode_policy.json",
	}

	for _, f := range files {
		var actual Policy

		err := Decode(&actual, testReadFile(t, f))
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("Input: %s\n\nActual: %#v\n\nExpected: %#v", f, actual, expected)
		}
	}
}

func TestDecode_structureMap(t *testing.T) {
	// This test is extracted from a failure in Terraform (terraform.io),
	// hence the interesting structure naming.

	type hclVariable struct {
		Default     interface{}
		Description string
		Fields      []string `hcl:",decodedFields"`
	}

	type rawConfig struct {
		Variable map[string]hclVariable
	}

	expected := rawConfig{
		Variable: map[string]hclVariable{
			"foo": hclVariable{
				Default:     "bar",
				Description: "bar",
				Fields:      []string{"Default", "Description"},
			},

			"amis": hclVariable{
				Default: []map[string]interface{}{
					map[string]interface{}{
						"east": "foo",
					},
				},
				Fields: []string{"Default"},
			},
		},
	}

	files := []string{
		"decode_tf_variable.hcl",
		"decode_tf_variable.json",
	}

	for _, f := range files {
		var actual rawConfig

		err := Decode(&actual, testReadFile(t, f))
		if err != nil {
			t.Fatalf("Input: %s\n\nerr: %s", f, err)
		}

		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("Input: %s\n\nActual: %#v\n\nExpected: %#v", f, actual, expected)
		}
	}
}
