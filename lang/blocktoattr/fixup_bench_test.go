package blocktoattr

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

func ambiguousAttrType(nesting int) cty.Type {
	if nesting == 0 {
		return cty.List(cty.Object(map[string]cty.Type{
			"a": cty.String,
			"b": cty.String,
		}))
	}
	return cty.List(cty.Object(map[string]cty.Type{
		"a":       cty.String,
		"b":       cty.String,
		"nested1": ambiguousAttrType(nesting - 1),
		"nested2": ambiguousAttrType(nesting - 1),
	}))
}

func schemaWithAmbiguousAttr(nesting int) *configschema.Block {
	return &configschema.Block{
		// This mimicks what the legacy SDK would return when a nested
		// *schema.Resource is set to use the "blocks as attributes" mode, which
		// requires fixup.
		Attributes: map[string]*configschema.Attribute{
			"maybe_block": {
				Type: ambiguousAttrType(nesting),
			},
		},
	}
}

func ambiguousNestedBlock(nesting int) *configschema.NestedBlock {
	ret := &configschema.NestedBlock{
		Nesting: configschema.NestingList,
		Block: configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"a": {Type: cty.String, Required: true},
				"b": {Type: cty.String, Required: true},
			},
		},
	}
	if nesting > 0 {
		ret.BlockTypes = map[string]*configschema.NestedBlock{
			"nested1": ambiguousNestedBlock(nesting - 1),
			"nested2": ambiguousNestedBlock(nesting - 1),
		}
	}
	return ret
}

func schemaWithoutAmbiguousAttr(nesting int) *configschema.Block {
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
}
`

func configBodyForFixupBlockAttrsBenchmark() hcl.Body {
	f, diags := hclsyntax.ParseConfig([]byte(configForFixupBlockAttrsBenchmark), "", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		panic("test configuration is invalid")
	}
	return f.Body
}

const benchmarkFixUpBlockAttrsNestingDepth = 4

func BenchmarkFixUpBlockAttrsWithAmbiguousAttr(b *testing.B) {
	body := configBodyForFixupBlockAttrsBenchmark()
	schema := schemaWithAmbiguousAttr(benchmarkFixUpBlockAttrsNestingDepth)
	if !MightNeedFixup(schema) {
		// Should never happen: this schema is designed to need fixup
		b.Fatal("test schema doesn't need fixup")
	}
	benchmarkFixUpBlockAttrs(b, body, schema)
}

func BenchmarkFixUpBlockAttrsWithoutAmbiguousAttr(b *testing.B) {
	body := configBodyForFixupBlockAttrsBenchmark()
	schema := schemaWithoutAmbiguousAttr(benchmarkFixUpBlockAttrsNestingDepth)
	if MightNeedFixup(schema) {
		// Should never happen: this schema is designed to be unambiguous
		b.Fatal("test schema might need fixup")
	}
	benchmarkFixUpBlockAttrs(b, body, schema)
}

func benchmarkFixUpBlockAttrs(b *testing.B, body hcl.Body, schema *configschema.Block) {
	spec := schema.DecoderSpec()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fixedBody := FixUpBlockAttrs(body, schema)
		val, diags := hcldec.Decode(fixedBody, spec, nil)
		if diags.HasErrors() {
			b.Fatal("diagnostics during decoding")
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
