package genconfig

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
)

func TestConfigGeneration(t *testing.T) {
	tcs := map[string]struct {
		schema   *configschema.Block
		addr     addrs.AbsResourceInstance
		provider addrs.LocalProviderConfig
		value    cty.Value
		expected string
	}{}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			contents, diags := GenerateResourceContents(tc.addr, tc.schema, tc.provider, tc.value)
			if len(diags) > 0 {
				t.Errorf("expected no diagnostics but found %s", diags)
			}

			got := WrapResourceContents(tc.addr, contents)
			if diff := cmp.Diff(got, tc.expected); len(diff) > 0 {
				t.Errorf("got:\n%s\nwant:\n%s\ndiff:\n%s", got, tc.expected, diff)
			}
		})
	}
}
