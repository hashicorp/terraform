package jsonconfig

import (
	"encoding/json"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcldec"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/lang"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// expression represents any unparsed expression
type expression struct {
	// "constant_value" is set only if the expression contains no references to
	// other objects, in which case it gives the resulting constant value. This
	// is mapped as for the individual values in the common value
	// representation.
	ConstantValue json.RawMessage `json:"constant_value,omitempty"`

	// Alternatively, "references" will be set to a list of references in the
	// expression. Multi-step references will be unwrapped and duplicated for
	// each significant traversal step, allowing callers to more easily
	// recognize the objects they care about without attempting to parse the
	// expressions. Callers should only use string equality checks here, since
	// the syntax may be extended in future releases.
	References []string `json:"references,omitempty"`
}

func marshalExpression(ex hcl.Expression) expression {
	var ret expression
	if ex != nil {
		val, _ := ex.Value(nil)
		if val != cty.NilVal {
			valJSON, _ := ctyjson.Marshal(val, val.Type())
			ret.ConstantValue = valJSON
		}
		vars, _ := lang.ReferencesInExpr(ex)
		var varString []string
		if len(vars) > 0 {
			for _, v := range vars {
				varString = append(varString, v.Subject.String())
			}
			ret.References = varString
		}
		return ret
	}
	return ret
}

func (e *expression) Empty() bool {
	return e.ConstantValue == nil && e.References == nil
}

// expressions is used to represent the entire content of a block. Attribute
// arguments are mapped directly with the attribute name as key and an
// expression as value.
type expressions map[string]interface{}

func marshalExpressions(body hcl.Body, schema *configschema.Block) expressions {
	// Since we want the raw, un-evaluated expressions we need to use the
	// low-level HCL API here, rather than the hcldec decoder API. That means we
	// need the low-level schema.
	lowSchema := hcldec.ImpliedSchema(schema.DecoderSpec())
	// (lowSchema is an hcl.BodySchema:
	// https://godoc.org/github.com/hashicorp/hcl2/hcl#BodySchema )

	// Use the low-level schema with the body to decode one level We'll just
	// ignore any additional content that's not covered by the schema, which
	// will effectively ignore "dynamic" blocks, and may also ignore other
	// unknown stuff but anything else would get flagged by Terraform as an
	// error anyway, and so we wouldn't end up in here.
	content, _, _ := body.PartialContent(lowSchema)
	if content == nil {
		// Should never happen for a valid body, but we'll just generate empty
		// if there were any problems.
		return nil
	}

	ret := make(expressions)

	// Any attributes we encode directly as expression objects.
	for name, attr := range content.Attributes {
		ret[name] = marshalExpression(attr.Expr) // note: singular expression for this one
	}

	// Any nested blocks require a recursive call to produce nested expressions
	// objects.
	for _, block := range content.Blocks {
		typeName := block.Type
		blockS, exists := schema.BlockTypes[typeName]
		if !exists {
			// Should never happen since only block types in the schema would be
			// put in blocks list
			continue
		}

		switch blockS.Nesting {
		case configschema.NestingSingle:
			ret[typeName] = marshalExpressions(block.Body, &blockS.Block)
		case configschema.NestingList, configschema.NestingSet:
			if _, exists := ret[typeName]; !exists {
				ret[typeName] = make([]map[string]interface{}, 0, 1)
			}
			ret[typeName] = append(ret[typeName].([]map[string]interface{}), marshalExpressions(block.Body, &blockS.Block))
		case configschema.NestingMap:
			if _, exists := ret[typeName]; !exists {
				ret[typeName] = make(map[string]map[string]interface{})
			}
			// NestingMap blocks always have the key in the first (and only) label
			key := block.Labels[0]
			retMap := ret[typeName].(map[string]map[string]interface{})
			retMap[key] = marshalExpressions(block.Body, &blockS.Block)
		}
	}

	return ret
}
