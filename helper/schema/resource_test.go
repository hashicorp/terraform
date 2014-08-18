package schema

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestResourceInternalValidate(t *testing.T) {
	cases := []struct {
		In  *Resource
		Err bool
	}{
		{
			nil,
			true,
		},

		// No optional and no required
		{
			&Resource{
				Schema: map[string]*Schema{
					"foo": &Schema{
						Type:     TypeInt,
						Optional: true,
						Required: true,
					},
				},
			},
			true,
		},
	}

	for i, tc := range cases {
		err := tc.In.InternalValidate()
		if (err != nil) != tc.Err {
			t.Fatalf("%d: bad: %s", i, err)
		}
	}
}

func TestResourceRefresh(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		if m != 42 {
			return fmt.Errorf("meta not passed")
		}

		return d.Set("foo", d.Get("foo").(int)+1)
	}

	s := &terraform.ResourceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	expected := &terraform.ResourceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo": "13",
		},
	}

	actual, err := r.Refresh(s, 42)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}
