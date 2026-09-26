// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type orderedField struct {
	Key   string
	Value any
}

type orderedObject []orderedField

func redactShowJSONBytes(input []byte) ([]byte, error) {
	if len(input) == 0 {
		return input, nil
	}

	rootValue, err := parseOrderedJSON(input)
	if err != nil {
		return nil, fmt.Errorf("failed to decode json for redaction: %w", err)
	}
	root, ok := asObject(rootValue)
	if !ok {
		return nil, fmt.Errorf("failed to decode json for redaction: root is not an object")
	}

	if valuesRaw, ok := getField(root, "values"); ok {
		if values, ok := asObject(valuesRaw); ok {
			redactStateValues(values, nil)
		}
	}

	sensitiveVars := map[string]struct{}{}
	sensitiveAttrs := map[string]map[string]struct{}{}

	if plannedValuesRaw, ok := getField(root, "planned_values"); ok {
		if plannedValues, ok := asObject(plannedValuesRaw); ok {
			redactStateValues(plannedValues, sensitiveAttrs)
		}
	}

	if priorStateRaw, ok := getField(root, "prior_state"); ok {
		if priorState, ok := asObject(priorStateRaw); ok {
			if valuesRaw, ok := getField(priorState, "values"); ok {
				if values, ok := asObject(valuesRaw); ok {
					redactStateValues(values, nil)
				}
			}
		}
	}

	if outputChangesRaw, ok := getField(root, "output_changes"); ok {
		if outputChanges, ok := asObject(outputChangesRaw); ok {
			for _, oc := range outputChanges {
				change, ok := asObject(oc.Value)
				if !ok {
					continue
				}
				setField(change, "before", redactByMask(getFieldOrNil(change, "before"), getFieldOrNil(change, "before_sensitive")))
				setField(change, "after", redactByMask(getFieldOrNil(change, "after"), getFieldOrNil(change, "after_sensitive")))
			}
		}
	}

	if resourceChangesRaw, ok := getField(root, "resource_changes"); ok {
		if resourceChanges, ok := asSlice(resourceChangesRaw); ok {
			for _, rcValue := range resourceChanges {
				rc, ok := asObject(rcValue)
				if !ok {
					continue
				}
				configAddr := resourceAddressToConfigAddress(getStringField(rc, "address"))
				changeRaw, ok := getField(rc, "change")
				if !ok {
					continue
				}
				change, ok := asObject(changeRaw)
				if !ok {
					continue
				}
				beforeMask := getFieldOrNil(change, "before_sensitive")
				afterMask := getFieldOrNil(change, "after_sensitive")
				setField(change, "before", redactByMask(getFieldOrNil(change, "before"), beforeMask))
				setField(change, "after", redactByMask(getFieldOrNil(change, "after"), afterMask))
				collectSensitiveAttrsFromMask(configAddr, beforeMask, sensitiveAttrs)
				collectSensitiveAttrsFromMask(configAddr, afterMask, sensitiveAttrs)
			}
		}
	}

	if deferredChangesRaw, ok := getField(root, "deferred_changes"); ok {
		if deferredChanges, ok := asSlice(deferredChangesRaw); ok {
			for _, dcValue := range deferredChanges {
				dc, ok := asObject(dcValue)
				if !ok {
					continue
				}
				rcRaw, ok := getField(dc, "resource_change")
				if !ok {
					continue
				}
				rc, ok := asObject(rcRaw)
				if !ok {
					continue
				}
				configAddr := resourceAddressToConfigAddress(getStringField(rc, "address"))
				changeRaw, ok := getField(rc, "change")
				if !ok {
					continue
				}
				change, ok := asObject(changeRaw)
				if !ok {
					continue
				}
				beforeMask := getFieldOrNil(change, "before_sensitive")
				afterMask := getFieldOrNil(change, "after_sensitive")
				setField(change, "before", redactByMask(getFieldOrNil(change, "before"), beforeMask))
				setField(change, "after", redactByMask(getFieldOrNil(change, "after"), afterMask))
				collectSensitiveAttrsFromMask(configAddr, beforeMask, sensitiveAttrs)
				collectSensitiveAttrsFromMask(configAddr, afterMask, sensitiveAttrs)
			}
		}
	}

	if actionInvocationsRaw, ok := getField(root, "action_invocations"); ok {
		if actionInvocations, ok := asSlice(actionInvocationsRaw); ok {
			for _, aiValue := range actionInvocations {
				ai, ok := asObject(aiValue)
				if !ok {
					continue
				}
				setField(ai, "config_values", redactByMask(getFieldOrNil(ai, "config_values"), getFieldOrNil(ai, "config_sensitive")))
			}
		}
	}

	if deferredAIsRaw, ok := getField(root, "deferred_action_invocations"); ok {
		if deferredAIs, ok := asSlice(deferredAIsRaw); ok {
			for _, daiValue := range deferredAIs {
				dai, ok := asObject(daiValue)
				if !ok {
					continue
				}
				aiRaw, ok := getField(dai, "action_invocation")
				if !ok {
					continue
				}
				ai, ok := asObject(aiRaw)
				if !ok {
					continue
				}
				setField(ai, "config_values", redactByMask(getFieldOrNil(ai, "config_values"), getFieldOrNil(ai, "config_sensitive")))
			}
		}
	}

	if configurationRaw, ok := getField(root, "configuration"); ok {
		if configuration, ok := asObject(configurationRaw); ok {
			if rootModuleRaw, ok := getField(configuration, "root_module"); ok {
				if rootModule, ok := asObject(rootModuleRaw); ok {
					redactConfigurationModule(rootModule, sensitiveVars, sensitiveAttrs)
				}
			}
		}
	}

	if variablesRaw, ok := getField(root, "variables"); ok {
		if variables, ok := asObject(variablesRaw); ok {
			for name := range sensitiveVars {
				if variableValue, exists := getField(variables, name); exists {
					if variable, ok := asObject(variableValue); ok {
						setField(variable, "value", nil)
					}
				}
			}
		}
	}

	return marshalOrderedJSON(root)
}

func redactStateValues(values orderedObject, collect map[string]map[string]struct{}) {
	if outputsRaw, ok := getField(values, "outputs"); ok {
		if outputs, ok := asObject(outputsRaw); ok {
			for _, ov := range outputs {
				output, ok := asObject(ov.Value)
				if !ok {
					continue
				}
				sensitiveValue, _ := getField(output, "sensitive")
				sensitive, _ := sensitiveValue.(bool)
				if sensitive {
					setField(output, "value", nil)
				}
			}
		}
	}
	if rootModuleRaw, ok := getField(values, "root_module"); ok {
		if rootModule, ok := asObject(rootModuleRaw); ok {
			redactStateModule(rootModule, collect)
		}
	}
}

func redactStateModule(module orderedObject, collect map[string]map[string]struct{}) {
	if resourcesRaw, ok := getField(module, "resources"); ok {
		if resources, ok := asSlice(resourcesRaw); ok {
			for _, rv := range resources {
				r, ok := asObject(rv)
				if !ok {
					continue
				}
				setField(r, "values", redactByMask(getFieldOrNil(r, "values"), getFieldOrNil(r, "sensitive_values")))
				if collect != nil {
					collectSensitiveAttrsFromMask(resourceAddressToConfigAddress(getStringField(r, "address")), getFieldOrNil(r, "sensitive_values"), collect)
				}
			}
		}
	}
	if childModulesRaw, ok := getField(module, "child_modules"); ok {
		if childModules, ok := asSlice(childModulesRaw); ok {
			for _, cv := range childModules {
				if child, ok := asObject(cv); ok {
					redactStateModule(child, collect)
				}
			}
		}
	}
}

func collectSensitiveAttrsFromMask(resourceAddr string, mask any, target map[string]map[string]struct{}) {
	if resourceAddr == "" {
		return
	}
	maskObj, ok := asObject(mask)
	if !ok {
		return
	}
	for _, attr := range maskObj {
		if !maskContainsSensitive(attr.Value) {
			continue
		}
		if _, exists := target[resourceAddr]; !exists {
			target[resourceAddr] = map[string]struct{}{}
		}
		target[resourceAddr][attr.Key] = struct{}{}
	}
}

func maskContainsSensitive(mask any) bool {
	switch m := mask.(type) {
	case bool:
		return m
	case orderedObject:
		for _, child := range m {
			if maskContainsSensitive(child.Value) {
				return true
			}
		}
	case []any:
		for _, child := range m {
			if maskContainsSensitive(child) {
				return true
			}
		}
	}
	return false
}

func redactConfigurationModule(module orderedObject, sensitiveVars map[string]struct{}, sensitiveAttrs map[string]map[string]struct{}) {
	if variablesRaw, ok := getField(module, "variables"); ok {
		if variables, ok := asObject(variablesRaw); ok {
			for _, variableField := range variables {
				variable, ok := asObject(variableField.Value)
				if !ok {
					continue
				}
				sensitiveValue, _ := getField(variable, "sensitive")
				sensitive, _ := sensitiveValue.(bool)
				if sensitive {
					sensitiveVars[variableField.Key] = struct{}{}
				}
				if _, shouldRedact := sensitiveVars[variableField.Key]; shouldRedact {
					if _, hasDefault := getField(variable, "default"); hasDefault {
						setField(variable, "default", nil)
					}
				}
			}
		}
	}

	if resourcesRaw, ok := getField(module, "resources"); ok {
		if resources, ok := asSlice(resourcesRaw); ok {
			for _, resourceValue := range resources {
				resource, ok := asObject(resourceValue)
				if !ok {
					continue
				}
				attrs := sensitiveAttrs[getStringField(resource, "address")]
				if len(attrs) == 0 {
					continue
				}
				expressionsRaw, ok := getField(resource, "expressions")
				if !ok {
					continue
				}
				expressions, ok := asObject(expressionsRaw)
				if !ok {
					continue
				}
				for attrName := range attrs {
					expressionValue, exists := getField(expressions, attrName)
					if !exists {
						continue
					}
					expression, ok := asObject(expressionValue)
					if !ok {
						continue
					}
					if _, hasConst := getField(expression, "constant_value"); hasConst {
						setField(expression, "constant_value", nil)
					}
				}
			}
		}
	}

	if moduleCallsRaw, ok := getField(module, "module_calls"); ok {
		if moduleCalls, ok := asObject(moduleCallsRaw); ok {
			for _, callValue := range moduleCalls {
				call, ok := asObject(callValue.Value)
				if !ok {
					continue
				}
				childRaw, ok := getField(call, "module")
				if !ok {
					continue
				}
				child, ok := asObject(childRaw)
				if !ok {
					continue
				}
				redactConfigurationModule(child, sensitiveVars, sensitiveAttrs)
			}
		}
	}
}

func redactByMask(value any, mask any) any {
	switch m := mask.(type) {
	case bool:
		if m {
			return nil
		}
		return value
	case orderedObject:
		valueObj, ok := asObject(value)
		if !ok {
			return value
		}
		for _, child := range m {
			if childValue, exists := getField(valueObj, child.Key); exists {
				setField(valueObj, child.Key, redactByMask(childValue, child.Value))
			}
		}
		return valueObj
	case []any:
		valueSlice, ok := asSlice(value)
		if !ok {
			return value
		}
		for i, childMask := range m {
			if i < len(valueSlice) {
				valueSlice[i] = redactByMask(valueSlice[i], childMask)
			}
		}
		return valueSlice
	default:
		return value
	}
}

func resourceAddressToConfigAddress(address string) string {
	if !strings.HasSuffix(address, "]") {
		return address
	}
	open := strings.LastIndex(address, "[")
	if open == -1 {
		return address
	}
	return address[:open]
}

func parseOrderedJSON(input []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(input))
	dec.UseNumber()

	v, err := decodeOrderedJSONValue(dec)
	if err != nil {
		return nil, err
	}

	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("trailing data after json value")
		}
		return nil, err
	}

	return v, nil
}

func decodeOrderedJSONValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}

	if delim, ok := tok.(json.Delim); ok {
		switch delim {
		case '{':
			obj := orderedObject{}
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyTok.(string)
				if !ok {
					return nil, fmt.Errorf("invalid object key token %T", keyTok)
				}
				value, err := decodeOrderedJSONValue(dec)
				if err != nil {
					return nil, err
				}
				obj = append(obj, orderedField{Key: key, Value: value})
			}
			endTok, err := dec.Token()
			if err != nil {
				return nil, err
			}
			if endDelim, ok := endTok.(json.Delim); !ok || endDelim != '}' {
				return nil, fmt.Errorf("invalid object terminator %v", endTok)
			}
			return obj, nil
		case '[':
			arr := []any{}
			for dec.More() {
				value, err := decodeOrderedJSONValue(dec)
				if err != nil {
					return nil, err
				}
				arr = append(arr, value)
			}
			endTok, err := dec.Token()
			if err != nil {
				return nil, err
			}
			if endDelim, ok := endTok.(json.Delim); !ok || endDelim != ']' {
				return nil, fmt.Errorf("invalid array terminator %v", endTok)
			}
			return arr, nil
		default:
			return nil, fmt.Errorf("unsupported delimiter %q", delim)
		}
	}

	return tok, nil
}

func marshalOrderedJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeOrderedJSONValue(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeOrderedJSONValue(buf *bytes.Buffer, v any) error {
	switch tv := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if tv {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case string:
		b, err := json.Marshal(tv)
		if err != nil {
			return err
		}
		buf.Write(b)
	case json.Number:
		buf.WriteString(tv.String())
	case float64, float32, int, int32, int64, uint, uint32, uint64:
		b, err := json.Marshal(tv)
		if err != nil {
			return err
		}
		buf.Write(b)
	case orderedObject:
		buf.WriteByte('{')
		for i, field := range tv {
			if i > 0 {
				buf.WriteByte(',')
			}
			keyBytes, err := json.Marshal(field.Key)
			if err != nil {
				return err
			}
			buf.Write(keyBytes)
			buf.WriteByte(':')
			if err := writeOrderedJSONValue(buf, field.Value); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	case []any:
		buf.WriteByte('[')
		for i, elem := range tv {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeOrderedJSONValue(buf, elem); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	default:
		b, err := json.Marshal(tv)
		if err != nil {
			return err
		}
		buf.Write(b)
	}
	return nil
}

func getField(obj orderedObject, key string) (any, bool) {
	for _, field := range obj {
		if field.Key == key {
			return field.Value, true
		}
	}
	return nil, false
}

func getStringField(obj orderedObject, key string) string {
	v, ok := getField(obj, key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func getFieldOrNil(obj orderedObject, key string) any {
	v, _ := getField(obj, key)
	return v
}

func setField(obj orderedObject, key string, value any) {
	for i := range obj {
		if obj[i].Key == key {
			obj[i].Value = value
			return
		}
	}
}

func asObject(v any) (orderedObject, bool) {
	obj, ok := v.(orderedObject)
	return obj, ok
}

func asSlice(v any) ([]any, bool) {
	s, ok := v.([]any)
	return s, ok
}
