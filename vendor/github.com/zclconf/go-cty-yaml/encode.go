package yaml

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/zclconf/go-cty/cty"
)

func (c *Converter) marshal(v cty.Value) ([]byte, error) {
	var buf bytes.Buffer

	e := &yaml_emitter_t{}
	yaml_emitter_initialize(e)
	yaml_emitter_set_output_writer(e, &buf)
	yaml_emitter_set_unicode(e, true)

	var evt yaml_event_t
	yaml_stream_start_event_initialize(&evt, yaml_UTF8_ENCODING)
	if !yaml_emitter_emit(e, &evt) {
		return nil, emitterError(e)
	}
	yaml_document_start_event_initialize(&evt, nil, nil, true)
	if !yaml_emitter_emit(e, &evt) {
		return nil, emitterError(e)
	}

	if err := c.marshalEmit(v, e); err != nil {
		return nil, err
	}

	yaml_document_end_event_initialize(&evt, true)
	if !yaml_emitter_emit(e, &evt) {
		return nil, emitterError(e)
	}
	yaml_stream_end_event_initialize(&evt)
	if !yaml_emitter_emit(e, &evt) {
		return nil, emitterError(e)
	}

	return buf.Bytes(), nil
}

func (c *Converter) marshalEmit(v cty.Value, e *yaml_emitter_t) error {
	ty := v.Type()
	switch {
	case v.IsNull():
		return c.marshalPrimitive(v, e)
	case !v.IsKnown():
		return fmt.Errorf("cannot serialize unknown value as YAML")
	case ty.IsPrimitiveType():
		return c.marshalPrimitive(v, e)
	case ty.IsTupleType(), ty.IsListType(), ty.IsSetType():
		return c.marshalSequence(v, e)
	case ty.IsObjectType(), ty.IsMapType():
		return c.marshalMapping(v, e)
	default:
		return fmt.Errorf("can't marshal %s as YAML", ty.FriendlyName())
	}
}

func (c *Converter) marshalPrimitive(v cty.Value, e *yaml_emitter_t) error {
	var evt yaml_event_t

	if v.IsNull() {
		yaml_scalar_event_initialize(
			&evt,
			nil,
			nil,
			[]byte("null"),
			true,
			true,
			yaml_PLAIN_SCALAR_STYLE,
		)
		if !yaml_emitter_emit(e, &evt) {
			return emitterError(e)
		}
		return nil
	}

	switch v.Type() {
	case cty.String:
		str := v.AsString()
		style := yaml_DOUBLE_QUOTED_SCALAR_STYLE
		if strings.Contains(str, "\n") {
			style = yaml_LITERAL_SCALAR_STYLE
		}
		yaml_scalar_event_initialize(
			&evt,
			nil,
			nil,
			[]byte(str),
			true,
			true,
			style,
		)
	case cty.Number:
		str := v.AsBigFloat().Text('f', -1)
		switch v {
		case cty.PositiveInfinity:
			str = "+.Inf"
		case cty.NegativeInfinity:
			str = "-.Inf"
		}
		yaml_scalar_event_initialize(
			&evt,
			nil,
			nil,
			[]byte(str),
			true,
			true,
			yaml_PLAIN_SCALAR_STYLE,
		)
	case cty.Bool:
		var str string
		switch v {
		case cty.True:
			str = "true"
		case cty.False:
			str = "false"
		}
		yaml_scalar_event_initialize(
			&evt,
			nil,
			nil,
			[]byte(str),
			true,
			true,
			yaml_PLAIN_SCALAR_STYLE,
		)
	}
	if !yaml_emitter_emit(e, &evt) {
		return emitterError(e)
	}
	return nil
}

func (c *Converter) marshalSequence(v cty.Value, e *yaml_emitter_t) error {
	style := yaml_BLOCK_SEQUENCE_STYLE
	if c.encodeAsFlow {
		style = yaml_FLOW_SEQUENCE_STYLE
	}

	var evt yaml_event_t
	yaml_sequence_start_event_initialize(&evt, nil, nil, true, style)
	if !yaml_emitter_emit(e, &evt) {
		return emitterError(e)
	}

	for it := v.ElementIterator(); it.Next(); {
		_, v := it.Element()
		err := c.marshalEmit(v, e)
		if err != nil {
			return err
		}
	}

	yaml_sequence_end_event_initialize(&evt)
	if !yaml_emitter_emit(e, &evt) {
		return emitterError(e)
	}
	return nil
}

func (c *Converter) marshalMapping(v cty.Value, e *yaml_emitter_t) error {
	style := yaml_BLOCK_MAPPING_STYLE
	if c.encodeAsFlow {
		style = yaml_FLOW_MAPPING_STYLE
	}

	var evt yaml_event_t
	yaml_mapping_start_event_initialize(&evt, nil, nil, true, style)
	if !yaml_emitter_emit(e, &evt) {
		return emitterError(e)
	}

	for it := v.ElementIterator(); it.Next(); {
		k, v := it.Element()
		err := c.marshalEmit(k, e)
		if err != nil {
			return err
		}
		err = c.marshalEmit(v, e)
		if err != nil {
			return err
		}
	}

	yaml_mapping_end_event_initialize(&evt)
	if !yaml_emitter_emit(e, &evt) {
		return emitterError(e)
	}
	return nil
}
