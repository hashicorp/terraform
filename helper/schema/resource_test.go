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
	}

	for i, tc := range cases {
		err := tc.In.InternalValidate()
		if (err != nil) != tc.Err {
			t.Fatalf("%d: bad: %s", i, err)
		}
	}
}
