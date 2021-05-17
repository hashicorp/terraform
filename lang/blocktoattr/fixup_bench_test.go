package blocktoattr

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

func ambiguousNestedBlock(nesting int) *configschema.NestedBlock {
	ret := &configschema.NestedBlock{
		Nesting: configschema.NestingList,
		Block: configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"a": {Type: cty.String, Required: true},
				"b": {Type: cty.String, Optional: true},
			},
		},
	}
	if nesting > 0 {
		ret.BlockTypes = map[string]*configschema.NestedBlock{
			"nested0": ambiguousNestedBlock(nesting - 1),
			"nested1": ambiguousNestedBlock(nesting - 1),
			"nested2": ambiguousNestedBlock(nesting - 1),
			"nested3": ambiguousNestedBlock(nesting - 1),
			"nested4": ambiguousNestedBlock(nesting - 1),
			"nested5": ambiguousNestedBlock(nesting - 1),
			"nested6": ambiguousNestedBlock(nesting - 1),
			"nested7": ambiguousNestedBlock(nesting - 1),
			"nested8": ambiguousNestedBlock(nesting - 1),
			"nested9": ambiguousNestedBlock(nesting - 1),
		}
	}
	return ret
}

func schemaWithAmbiguousNestedBlock(nesting int) *configschema.Block {
	return &configschema.Block{
		BlockTypes: map[string]*configschema.NestedBlock{
			"maybe_block": ambiguousNestedBlock(nesting),
		},
	}
}

const configForFixupBlockAttrsBenchmark = `
maybe_block {
  a = "hello"
  b = "world"
  nested0 {
    a = "the"
    nested1 {
	  a = "deeper"
      nested2 {
        a = "we"
        nested3 {
          a = "go"
          b = "inside"
        }
      }
    }
  }
}
`

func configBodyForFixupBlockAttrsBenchmark() hcl.Body {
	f, diags := hclsyntax.ParseConfig([]byte(configForFixupBlockAttrsBenchmark), "", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		panic("test configuration is invalid")
	}
	return f.Body
}

func BenchmarkFixUpBlockAttrs(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		body := configBodyForFixupBlockAttrsBenchmark()
		schema := schemaWithAmbiguousNestedBlock(5)
		b.StartTimer()

		spec := schema.DecoderSpec()
		fixedBody := FixUpBlockAttrs(body, schema)
		val, diags := hcldec.Decode(fixedBody, spec, nil)
		if diags.HasErrors() {
			b.Fatal("diagnostics during decoding", diags)
		}
		if !val.Type().IsObjectType() {
			b.Fatal("result is not an object")
		}
		blockVal := val.GetAttr("maybe_block")
		if !blockVal.Type().IsListType() || blockVal.LengthInt() != 1 {
			b.Fatal("result has wrong value for 'maybe_block'")
		}
	}
}
