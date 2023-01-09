package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
)

func (v Value) computeAttributeChangeAsObject(attributes map[string]cty.Type) change.Change {
	var keys []string
	for key := range attributes {
		keys = append(keys, key)
	}

	attributeChanges, changeType := v.processObject(keys, func(key string) interface{} {
		return attributes[key]
	})
	return change.New(change.Object(attributeChanges), changeType, v.replacePath())
}

func (v Value) computeAttributeChangeAsNestedObject(attributes map[string]*jsonprovider.Attribute) change.Change {
	var keys []string
	for key := range attributes {
		keys = append(keys, key)
	}

	attributeChanges, changeType := v.processObject(keys, func(key string) interface{} {
		return attributes[key]
	})
	return change.New(change.NestedObject(attributeChanges), changeType, v.replacePath())
}

func (v Value) processObject(keys []string, getAttribute func(string) interface{}) (map[string]change.Change, plans.Action) {
	attributeChanges := make(map[string]change.Change)
	mapValue := v.asMap()

	currentAction := v.getDefaultActionForIteration()
	for _, key := range keys {
		attribute := getAttribute(key)
		attributeValue := mapValue.getChild(key)

		// We always assume changes to object are implicit.
		attributeValue.BeforeExplicit = false
		attributeValue.AfterExplicit = false

		// We use the generic ComputeChange here, as we don't know whether this
		// is from a nested object or a `normal` object.
		attributeChange := attributeValue.ComputeChange(attribute)
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
