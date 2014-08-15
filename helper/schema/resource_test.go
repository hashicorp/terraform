package schema

import (
	"testing"
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

		// Missing Type
		{
			&Resource{
				Schema: map[string]*Schema{
					"foo": &Schema{
						Required: true,
					},
				},
			},
			true,
		},

		// Required but computed
		{
			&Resource{
				Schema: map[string]*Schema{
					"foo": &Schema{
						Type:     TypeInt,
						Required: true,
						Computed: true,
					},
				},
			},
			true,
		},

		// Looks good
		{
			&Resource{
				Schema: map[string]*Schema{
					"foo": &Schema{
						Type:     TypeString,
						Required: true,
					},
				},
			},
			false,
		},

		// List element not set
		{
			&Resource{
				Schema: map[string]*Schema{
					"foo": &Schema{
						Type: TypeList,
					},
				},
			},
			true,
		},

		// List element computed
		{
			&Resource{
				Schema: map[string]*Schema{
					"foo": &Schema{
						Type: TypeList,
						Elem: &Schema{
							Type:     TypeInt,
							Computed: true,
						},
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
