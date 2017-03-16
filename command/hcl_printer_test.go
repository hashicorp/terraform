package command

import "testing"

// The command package has it's own HCL encoder to encode variables to push.
// Make sure the variable we encode parse correctly
func TestHCLEncoder_parse(t *testing.T) {
	cases := []struct {
		Name  string
		Val   interface{}
		Error bool
	}{
		{
			Name: "int",
			Val:  12345,
		},
		{
			Name: "float",
			Val:  1.2345,
		},
		{
			Name: "string",
			Val:  "terraform",
		},
		{
			Name: "list",
			Val:  []interface{}{"a", "b", "c"},
		},
		{
			Name: "map",
			Val: map[string]interface{}{
				"a": 1,
			},
		},
		// a numeric looking identifier requires quotes
		{
			Name: "map_with_quoted_key",
			Val: map[string]interface{}{
				"0.0.0.0/24": "mask",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			_, err := encodeHCL(c.Val)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
