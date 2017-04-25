package config

import (
	"fmt"
	"testing"
)

func TestBackendHash(t *testing.T) {
	// WARNING: The codes below should _never_ change. If they change, it
	// means that a future TF version may falsely recognize unchanged backend
	// configuration as changed. Ultimately this should have no adverse
	// affect but it is annoying for users and should be avoided if possible.

	cases := []struct {
		Name    string
		Fixture string
		Code    uint64
	}{
		{
			"no backend config",
			"backend-hash-empty",
			0,
		},

		{
			"backend config with only type",
			"backend-hash-type-only",
			17852588448730441876,
		},

		{
			"backend config with type and config",
			"backend-hash-basic",
			10288498853650209002,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			c := testConfig(t, tc.Fixture)
			err := c.Validate()
			if err != nil {
				t.Fatalf("err: %s", err)
			}

			var actual uint64
			if c.Terraform != nil && c.Terraform.Backend != nil {
				actual = c.Terraform.Backend.Hash
			}
			if actual != tc.Code {
				t.Fatalf("bad: %d != %d", actual, tc.Code)
			}
		})
	}
}
