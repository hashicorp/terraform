package schema

import (
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

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
				Resources: map[string]*Resource{
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
		if (len(es) > 0) != tc.Err {
			t.Fatalf("%d: %#v", i, es)
		}
	}
}
