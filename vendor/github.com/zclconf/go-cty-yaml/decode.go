package yaml

import (
	"errors"
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

func (c *Converter) unmarshal(src []byte, ty cty.Type) (cty.Value, error) {
	p := &yaml_parser_t{}
	if !yaml_parser_initialize(p) {
		return cty.NilVal, errors.New("failed to initialize YAML parser")
	}
	if len(src) == 0 {
		src = []byte{'\n'}
	}

	an := &valueAnalysis{
		anchorsPending: map[string]int{},
		anchorVals:     map[string]cty.Value{},
	}

	yaml_parser_set_input_string(p, src)

	var evt yaml_event_t
	if !yaml_parser_parse(p, &evt) {
		return cty.NilVal, parserError(p)
	}
	if evt.typ != yaml_STREAM_START_EVENT {
		return cty.NilVal, parseEventErrorf(&evt, "missing stream start token")
	}
	if !yaml_parser_parse(p, &evt) {
		return cty.NilVal, parserError(p)
	}
	if evt.typ != yaml_DOCUMENT_START_EVENT {
		return cty.NilVal, parseEventErrorf(&evt, "missing start of document")
	}

	v, err := c.unmarshalParse(an, p)
	if err != nil {
		return cty.NilVal, err
	}

	if !yaml_parser_parse(p, &evt) {
		return cty.NilVal, parserError(p)
	}
	if evt.typ == yaml_DOCUMENT_START_EVENT {
		return cty.NilVal, parseEventErrorf(&evt, "only a single document is allowed")
	}
	if evt.typ != yaml_DOCUMENT_END_EVENT {
		return cty.NilVal, parseEventErrorf(&evt, "unexpected extra content (%s) after value", evt.typ.String())
	}
	if !yaml_parser_parse(p, &evt) {
		return cty.NilVal, parserError(p)
	}
	if evt.typ != yaml_STREAM_END_EVENT {
		return cty.NilVal, parseEventErrorf(&evt, "unexpected extra content after value")
	}

	return convert.Convert(v, ty)
}

func (c *Converter) unmarshalParse(an *valueAnalysis, p *yaml_parser_t) (cty.Value, error) {
	var evt yaml_event_t
	if !yaml_parser_parse(p, &evt) {
		return cty.NilVal, parserError(p)
	}
	return c.unmarshalParseRemainder(an, &evt, p)
}

func (c *Converter) unmarshalParseRemainder(an *valueAnalysis, evt *yaml_event_t, p *yaml_parser_t) (cty.Value, error) {
	switch evt.typ {
	case yaml_SCALAR_EVENT:
		return c.unmarshalScalar(an, evt, p)
	case yaml_ALIAS_EVENT:
		return c.unmarshalAlias(an, evt, p)
	case yaml_MAPPING_START_EVENT:
		return c.unmarshalMapping(an, evt, p)
	case yaml_SEQUENCE_START_EVENT:
		return c.unmarshalSequence(an, evt, p)
	case yaml_DOCUMENT_START_EVENT:
		return cty.NilVal, parseEventErrorf(evt, "only a single document is allowed")
	case yaml_STREAM_END_EVENT:
		// Decoding an empty buffer, probably
		return cty.NilVal, parseEventErrorf(evt, "expecting value but found end of stream")
	default:
		// Should never happen; the above should be comprehensive
		return cty.NilVal, parseEventErrorf(evt, "unexpected parser event %s", evt.typ.String())
	}
}

func (c *Converter) unmarshalScalar(an *valueAnalysis, evt *yaml_event_t, p *yaml_parser_t) (cty.Value, error) {
	src := evt.value
	tag := string(evt.tag)
	anchor := string(evt.anchor)

	if len(anchor) > 0 {
		an.beginAnchor(anchor)
	}

	val, err := c.resolveScalar(tag, string(src), yaml_scalar_style_t(evt.style))
	if err != nil {
		return cty.NilVal, parseEventErrorWrap(evt, err)
	}

	if val.RawEquals(mergeMappingVal) {
		// In any context other than a mapping key, this is just a plain string
		val = cty.StringVal("<<")
	}

	if len(anchor) > 0 {
		an.completeAnchor(anchor, val)
	}
	return val, nil
}

func (c *Converter) unmarshalMapping(an *valueAnalysis, evt *yaml_event_t, p *yaml_parser_t) (cty.Value, error) {
	tag := string(evt.tag)
	anchor := string(evt.anchor)

	if tag != "" && tag != yaml_MAP_TAG {
		return cty.NilVal, parseEventErrorf(evt, "can't interpret mapping as %s", tag)
	}

	if anchor != "" {
		an.beginAnchor(anchor)
	}

	vals := make(map[string]cty.Value)
	for {
		var nextEvt yaml_event_t
		if !yaml_parser_parse(p, &nextEvt) {
			return cty.NilVal, parserError(p)
		}
		if nextEvt.typ == yaml_MAPPING_END_EVENT {
			v := cty.ObjectVal(vals)
			if anchor != "" {
				an.completeAnchor(anchor, v)
			}
			return v, nil
		}

		if nextEvt.typ != yaml_SCALAR_EVENT {
			return cty.NilVal, parseEventErrorf(&nextEvt, "only strings are allowed as mapping keys")
		}
		keyVal, err := c.resolveScalar(string(nextEvt.tag), string(nextEvt.value), yaml_scalar_style_t(nextEvt.style))
		if err != nil {
			return cty.NilVal, err
		}
		if keyVal.RawEquals(mergeMappingVal) {
			// Merging the value (which must be a mapping) into our mapping,
			// then.
			val, err := c.unmarshalParse(an, p)
			if err != nil {
				return cty.NilVal, err
			}
			ty := val.Type()
			if !(ty.IsObjectType() || ty.IsMapType()) {
				return cty.NilVal, parseEventErrorf(&nextEvt, "cannot merge %s into mapping", ty.FriendlyName())
			}
			for it := val.ElementIterator(); it.Next(); {
				k, v := it.Element()
				vals[k.AsString()] = v
			}
			continue
		}
		if keyValStr, err := convert.Convert(keyVal, cty.String); err == nil {
			keyVal = keyValStr
		} else {
			return cty.NilVal, parseEventErrorf(&nextEvt, "only strings are allowed as mapping keys")
		}
		if keyVal.IsNull() {
			return cty.NilVal, parseEventErrorf(&nextEvt, "mapping key cannot be null")
		}
		if !keyVal.IsKnown() {
			return cty.NilVal, parseEventErrorf(&nextEvt, "mapping key must be known")
		}
		val, err := c.unmarshalParse(an, p)
		if err != nil {
			return cty.NilVal, err
		}

		vals[keyVal.AsString()] = val
	}
}

func (c *Converter) unmarshalSequence(an *valueAnalysis, evt *yaml_event_t, p *yaml_parser_t) (cty.Value, error) {
	tag := string(evt.tag)
	anchor := string(evt.anchor)

	if tag != "" && tag != yaml_SEQ_TAG {
		return cty.NilVal, parseEventErrorf(evt, "can't interpret sequence as %s", tag)
	}

	if anchor != "" {
		an.beginAnchor(anchor)
	}

	var vals []cty.Value
	for {
		var nextEvt yaml_event_t
		if !yaml_parser_parse(p, &nextEvt) {
			return cty.NilVal, parserError(p)
		}
		if nextEvt.typ == yaml_SEQUENCE_END_EVENT {
			ty := cty.TupleVal(vals)
			if anchor != "" {
				an.completeAnchor(anchor, ty)
			}
			return ty, nil
		}

		val, err := c.unmarshalParseRemainder(an, &nextEvt, p)
		if err != nil {
			return cty.NilVal, err
		}

		vals = append(vals, val)
	}
}

func (c *Converter) unmarshalAlias(an *valueAnalysis, evt *yaml_event_t, p *yaml_parser_t) (cty.Value, error) {
	v, err := an.anchorVal(string(evt.anchor))
	if err != nil {
		err = parseEventErrorWrap(evt, err)
	}
	return v, err
}

type valueAnalysis struct {
	anchorsPending map[string]int
	anchorVals     map[string]cty.Value
}

func (an *valueAnalysis) beginAnchor(name string) {
	an.anchorsPending[name]++
}

func (an *valueAnalysis) completeAnchor(name string, v cty.Value) {
	an.anchorsPending[name]--
	if an.anchorsPending[name] == 0 {
		delete(an.anchorsPending, name)
	}
	an.anchorVals[name] = v
}

func (an *valueAnalysis) anchorVal(name string) (cty.Value, error) {
	if _, pending := an.anchorsPending[name]; pending {
		// YAML normally allows self-referencing structures, but cty cannot
		// represent them (it requires all structures to be finite) so we
		// must fail here.
		return cty.NilVal, fmt.Errorf("cannot refer to anchor %q from inside its own definition", name)
	}
	ty, ok := an.anchorVals[name]
	if !ok {
		return cty.NilVal, fmt.Errorf("reference to undefined anchor %q", name)
	}
	return ty, nil
}
