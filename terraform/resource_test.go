package terraform

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config"
)

func TestInstanceInfo(t *testing.T) {
	cases := []struct {
		Info   *InstanceInfo
		Result string
	}{
		{
			&InstanceInfo{
				Id: "foo",
			},
			"foo",
		},
		{
			&InstanceInfo{
				Id:         "foo",
				ModulePath: rootModulePath,
			},
			"foo",
		},
		{
			&InstanceInfo{
				Id:         "foo",
				ModulePath: []string{"root", "consul"},
			},
			"module.consul.foo",
		},
	}

	for i, tc := range cases {
		actual := tc.Info.HumanId()
		if actual != tc.Result {
			t.Fatalf("%d: %s", i, actual)
		}
	}
}

func TestResourceConfigGet(t *testing.T) {
	cases := []struct {
		Config map[string]interface{}
		Vars   map[string]string
		Key    string
		Value  interface{}
	}{
		{
			Config: nil,
			Key:    "foo",
			Value:  nil,
		},

		{
			Config: map[string]interface{}{
				"foo": "${var.foo}",
			},
			Key:   "foo",
			Value: "${var.foo}",
		},

		{
			Config: map[string]interface{}{
				"foo": "${var.foo}",
			},
			Vars:  map[string]string{"foo": "bar"},
			Key:   "foo",
			Value: "bar",
		},

		{
			Config: map[string]interface{}{
				"foo": []interface{}{1, 2, 5},
			},
			Key:   "foo.0",
			Value: 1,
		},

		{
			Config: map[string]interface{}{
				"foo": []interface{}{1, 2, 5},
			},
			Key:   "foo.5",
			Value: nil,
		},
	}

	for i, tc := range cases {
		var rawC *config.RawConfig
		if tc.Config != nil {
			var err error
			rawC, err = config.NewRawConfig(tc.Config)
			if err != nil {
				t.Fatalf("err: %s", err)
			}
		}

		rc := NewResourceConfig(rawC)
		if tc.Vars != nil {
			ctx := NewContext(&ContextOpts{Variables: tc.Vars})
			err := rc.interpolate(
				ctx.walkContext(walkInvalid, rootModulePath),
				nil)
			if err != nil {
				t.Fatalf("err: %s", err)
			}
		}

		v, _ := rc.Get(tc.Key)
		if !reflect.DeepEqual(v, tc.Value) {
			t.Fatalf("%d bad: %#v", i, v)
		}
	}
}

func testResourceConfig(
	t *testing.T, c map[string]interface{}) *ResourceConfig {
	raw, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return NewResourceConfig(raw)
}
