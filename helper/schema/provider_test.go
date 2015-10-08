package schema

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = new(Provider)
}

func TestProviderConfigure(t *testing.T) {
	cases := []struct {
		P      *Provider
		Config map[string]interface{}
		Err    bool
	}{
		{
			P:      &Provider{},
			Config: nil,
			Err:    false,
		},

		{
			P: &Provider{
				Schema: map[string]*Schema{
					"foo": &Schema{
						Type:     TypeInt,
						Optional: true,
					},
				},

				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					if d.Get("foo").(int) == 42 {
						return nil, nil
					}

					return nil, fmt.Errorf("nope")
				},
			},
			Config: map[string]interface{}{
				"foo": 42,
			},
			Err: false,
		},

		{
			P: &Provider{
				Schema: map[string]*Schema{
					"foo": &Schema{
						Type:     TypeInt,
						Optional: true,
					},
				},

				ConfigureFunc: func(d *ResourceData) (interface{}, error) {
					if d.Get("foo").(int) == 42 {
						return nil, nil
					}

					return nil, fmt.Errorf("nope")
				},
			},
			Config: map[string]interface{}{
				"foo": 52,
			},
			Err: true,
		},
	}

	for i, tc := range cases {
		c, err := config.NewRawConfig(tc.Config)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		err = tc.P.Configure(terraform.NewResourceConfig(c))
		if err != nil != tc.Err {
			t.Fatalf("%d: %s", i, err)
		}
	}
}

func TestProviderResources(t *testing.T) {
	cases := []struct {
		P      *Provider
		Result []terraform.ResourceType
	}{
		{
			P:      &Provider{},
			Result: []terraform.ResourceType{},
		},

		{
			P: &Provider{
				ResourcesMap: map[string]*Resource{
					"foo": nil,
					"bar": nil,
				},
			},
			Result: []terraform.ResourceType{
				terraform.ResourceType{Name: "bar"},
				terraform.ResourceType{Name: "foo"},
			},
		},
	}

	for i, tc := range cases {
		actual := tc.P.Resources()
		if !reflect.DeepEqual(actual, tc.Result) {
			t.Fatalf("%d: %#v", i, actual)
		}
	}
}

func TestProviderValidate(t *testing.T) {
	cases := []struct {
		P      *Provider
		Config map[string]interface{}
		Err    bool
	}{
		{
			P: &Provider{
				Schema: map[string]*Schema{
					"foo": &Schema{},
				},
			},
			Config: nil,
			Err:    true,
		},
	}

	for i, tc := range cases {
		c, err := config.NewRawConfig(tc.Config)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		_, es := tc.P.Validate(terraform.NewResourceConfig(c))
		if len(es) > 0 != tc.Err {
			t.Fatalf("%d: %#v", i, es)
		}
	}
}

func TestProviderValidateResource(t *testing.T) {
	cases := []struct {
		P      *Provider
		Type   string
		Config map[string]interface{}
		Err    bool
	}{
		{
			P:      &Provider{},
			Type:   "foo",
			Config: nil,
			Err:    true,
		},

		{
			P: &Provider{
				ResourcesMap: map[string]*Resource{
					"foo": &Resource{},
				},
			},
			Type:   "foo",
			Config: nil,
			Err:    false,
		},
	}

	for i, tc := range cases {
		c, err := config.NewRawConfig(tc.Config)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		_, es := tc.P.ValidateResource(tc.Type, terraform.NewResourceConfig(c))
		if len(es) > 0 != tc.Err {
			t.Fatalf("%d: %#v", i, es)
		}
	}
}

func TestProviderMeta(t *testing.T) {
	p := new(Provider)
	if v := p.Meta(); v != nil {
		t.Fatalf("bad: %#v", v)
	}

	expected := 42
	p.SetMeta(42)
	if v := p.Meta(); !reflect.DeepEqual(v, expected) {
		t.Fatalf("bad: %#v", v)
	}
}
