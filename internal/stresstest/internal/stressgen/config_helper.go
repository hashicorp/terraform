package stressgen

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform/addrs"
	"github.com/zclconf/go-cty/cty/gocty"
)

// appendRepetitionMetaArgs inserts either a for_each argument, a count
// argument, or neither into the given body, and returns true if it inserted
// an argument.
//
// At least one of forEach or count must be nil, or this function will panic.
func appendRepetitionMetaArgs(body *hclwrite.Body, forEach *ConfigExprForEach, count *ConfigExprCount) bool {
	switch {
	case forEach != nil && count != nil:
		panic("for_each and count are mutually-exclusive")
	case forEach != nil:
		body.SetAttributeRaw("for_each", forEach.BuildExpr().BuildTokens(nil))
		return true
	case count != nil:
		body.SetAttributeRaw("count", count.BuildExpr().BuildTokens(nil))
		return true
	default:
		return false
	}
}

// instanceKeysForRepetitionMetaArgs takes a for_each expression, a count
// expression, or neither. It then generates a list of instance keys that
// Terraform ought to produce from the values of those expressions, as
// determined by the given registry.
//
// At least one of forEach or count must be nil, or this function will panic.
func instanceKeysForRepetitionMetaArgs(reg *Registry, forEach *ConfigExprForEach, count *ConfigExprCount) []addrs.InstanceKey {
	switch {
	case forEach != nil && count != nil:
		panic("for_each and count are mutually-exclusive")
	case forEach != nil:
		// Our instance keys are the keys from the expected value of for_each,
		// which should always be a mapping due to how ConfigExprForEach is
		// written.
		forEachVal := forEach.ExpectedValue(reg)
		var instanceKeys []addrs.InstanceKey
		for it := forEachVal.ElementIterator(); it.Next(); {
			k, _ := it.Element()
			instanceKeys = append(instanceKeys, addrs.StringKey(k.AsString()))
		}
		return instanceKeys
	case count != nil:
		countVal := count.ExpectedValue(reg)
		var n int
		err := gocty.FromCtyValue(countVal, &n)
		if err != nil {
			panic("count expression didn't produce an integer")
		}
		if n == 0 {
			return nil
		}
		instanceKeys := make([]addrs.InstanceKey, n)
		for i := range instanceKeys {
			instanceKeys[i] = addrs.IntKey(i)
		}
		return instanceKeys
	default:
		return noRepetitionInstanceKeys
	}
}

// When we're not doing repetition there's always exactly one instance key,
// representing the absense of an instance key.
var noRepetitionInstanceKeys = []addrs.InstanceKey{addrs.NoKey}
