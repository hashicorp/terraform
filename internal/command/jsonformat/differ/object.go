package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
)

func (v Value) computeAttributeChangeAsObject(attributes map[string]cty.Type) change.Change {
	attributeChanges, changeType := processObject(v, attributes, func(value Value, ctype cty.Type) change.Change {
		return value.computeChangeForType(ctype)
	})
	return change.New(change.Object(attributeChanges), changeType, v.replacePath())
}

func (v Value) computeAttributeChangeAsNestedObject(attributes map[string]*jsonprovider.Attribute) change.Change {
	attributeChanges, changeType := processObject(v, attributes, func(value Value, attribute *jsonprovider.Attribute) change.Change {
		return value.ComputeChangeForAttribute(attribute)
	})
	return change.New(change.NestedObject(attributeChanges), changeType, v.replacePath())
}

// processObject steps through the children of value as if it is an object and
// calls out to the provided computeChange function once it has collated the
// diffs for each child attribute.
//
// We have to make this generic as attributes and nested objects process either
// cty.Type or jsonprovider.Attribute children respectively. And we want to
// reuse as much code as possible.
//
// Also, as it generic we cannot make this function a method on Value as you
// can't create generic methods on structs. Instead, we make this a generic
// function that receives the value as an argument.
func processObject[T any](v Value, attributes map[string]T, computeChange func(Value, T) change.Change) (map[string]change.Change, plans.Action) {
	attributeChanges := make(map[string]change.Change)
	mapValue := v.asMap()

	currentAction := v.getDefaultActionForIteration()
	for key, attribute := range attributes {
		attributeValue := mapValue.getChild(key)

		// We always assume changes to object are implicit.
		attributeValue.BeforeExplicit = false
		attributeValue.AfterExplicit = false

		// We use the generic ComputeChange here, as we don't know whether this
		// is from a nested object or a `normal` object.
		attributeChange := computeChange(attributeValue, attribute)
		if attributeChange.GetAction() == plans.NoOp && attributeValue.Before == nil && attributeValue.After == nil {
			// We skip attributes of objects that are null both before and
			// after. We don't even count these as unchanged attributes.
			continue
		}
		attributeChanges[key] = attributeChange
		currentAction = compareActions(currentAction, attributeChange.GetAction())
	}

	return attributeChanges, currentAction
}
