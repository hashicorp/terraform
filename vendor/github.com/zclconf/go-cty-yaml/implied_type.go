package yaml

import (
	"errors"
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

func (c *Converter) impliedType(src []byte) (cty.Type, error) {
	p := &yaml_parser_t{}
	if !yaml_parser_initialize(p) {
		return cty.NilType, errors.New("failed to initialize YAML parser")
	}
	if len(src) == 0 {
		src = []byte{'\n'}
	}

	an := &typeAnalysis{
		anchorsPending: map[string]int{},
		anchorTypes:    map[string]cty.Type{},
	}

	yaml_parser_set_input_string(p, src)

	var evt yaml_event_t
	if !yaml_parser_parse(p, &evt) {
		return cty.NilType, parserError(p)
	}
	if evt.typ != yaml_STREAM_START_EVENT {
		return cty.NilType, parseEventErrorf(&evt, "missing stream start token")
	}
	if !yaml_parser_parse(p, &evt) {
		return cty.NilType, parserError(p)
	}
	if evt.typ != yaml_DOCUMENT_START_EVENT {
		return cty.NilType, parseEventErrorf(&evt, "missing start of document")
	}

	ty, err := c.impliedTypeParse(an, p)
	if err != nil {
		return cty.NilType, err
	}

	if !yaml_parser_parse(p, &evt) {
		return cty.NilType, parserError(p)
	}
	if evt.typ == yaml_DOCUMENT_START_EVENT {
		return cty.NilType, parseEventErrorf(&evt, "only a single document is allowed")
	}
	if evt.typ != yaml_DOCUMENT_END_EVENT {
		return cty.NilType, parseEventErrorf(&evt, "unexpected extra content (%s) after value", evt.typ.String())
	}
	if !yaml_parser_parse(p, &evt) {
		return cty.NilType, parserError(p)
	}
	if evt.typ != yaml_STREAM_END_EVENT {
		return cty.NilType, parseEventErrorf(&evt, "unexpected extra content after value")
	}

	return ty, err
}

func (c *Converter) impliedTypeParse(an *typeAnalysis, p *yaml_parser_t) (cty.Type, error) {
	var evt yaml_event_t
	if !yaml_parser_parse(p, &evt) {
		return cty.NilType, parserError(p)
	}
	return c.impliedTypeParseRemainder(an, &evt, p)
}

func (c *Converter) impliedTypeParseRemainder(an *typeAnalysis, evt *yaml_event_t, p *yaml_parser_t) (cty.Type, error) {
	switch evt.typ {
	case yaml_SCALAR_EVENT:
		return c.impliedTypeScalar(an, evt, p)
	case yaml_ALIAS_EVENT:
		return c.impliedTypeAlias(an, evt, p)
	case yaml_MAPPING_START_EVENT:
		return c.impliedTypeMapping(an, evt, p)
	case yaml_SEQUENCE_START_EVENT:
		return c.impliedTypeSequence(an, evt, p)
	case yaml_DOCUMENT_START_EVENT:
		return cty.NilType, parseEventErrorf(evt, "only a single document is allowed")
	case yaml_STREAM_END_EVENT:
		// Decoding an empty buffer, probably
		return cty.NilType, parseEventErrorf(evt, "expecting value but found end of stream")
	default:
		// Should never happen; the above should be comprehensive
		return cty.NilType, parseEventErrorf(evt, "unexpected parser event %s", evt.typ.String())
	}
}

func (c *Converter) impliedTypeScalar(an *typeAnalysis, evt *yaml_event_t, p *yaml_parser_t) (cty.Type, error) {
	src := evt.value
	tag := string(evt.tag)
	anchor := string(evt.anchor)
	implicit := evt.implicit

	if len(anchor) > 0 {
		an.beginAnchor(anchor)
	}

	var ty cty.Type
	switch {
	case tag == "" && !implicit:
		// Untagged explicit string
		ty = cty.String
	default:
		v, err := c.resolveScalar(tag, string(src), yaml_scalar_style_t(evt.style))
		if err != nil {
			return cty.NilType, parseEventErrorWrap(evt, err)
		}
		if v.RawEquals(mergeMappingVal) {
			// In any context other than a mapping key, this is just a plain string
			ty = cty.String
		} else {
			ty = v.Type()
		}
	}

	if len(anchor) > 0 {
		an.completeAnchor(anchor, ty)
	}
	return ty, nil
}

func (c *Converter) impliedTypeMapping(an *typeAnalysis, evt *yaml_event_t, p *yaml_parser_t) (cty.Type, error) {
	tag := string(evt.tag)
	anchor := string(evt.anchor)

	if tag != "" && tag != yaml_MAP_TAG {
		return cty.NilType, parseEventErrorf(evt, "can't interpret mapping as %s", tag)
	}

	if anchor != "" {
		an.beginAnchor(anchor)
	}

	atys := make(map[string]cty.Type)
	for {
		var nextEvt yaml_event_t
		if !yaml_parser_parse(p, &nextEvt) {
			return cty.NilType, parserError(p)
		}
		if nextEvt.typ == yaml_MAPPING_END_EVENT {
			ty := cty.Object(atys)
			if anchor != "" {
				an.completeAnchor(anchor, ty)
			}
			return ty, nil
		}

		if nextEvt.typ != yaml_SCALAR_EVENT {
			return cty.NilType, parseEventErrorf(&nextEvt, "only strings are allowed as mapping keys")
		}
		keyVal, err := c.resolveScalar(string(nextEvt.tag), string(nextEvt.value), yaml_scalar_style_t(nextEvt.style))
		if err != nil {
			return cty.NilType, err
		}
		if keyVal.RawEquals(mergeMappingVal) {
			// Merging the value (which must be a mapping) into our mapping,
			// then.
			ty, err := c.impliedTypeParse(an, p)
			if err != nil {
				return cty.NilType, err
			}
			if !ty.IsObjectType() {
				return cty.NilType, parseEventErrorf(&nextEvt, "cannot merge %s into mapping", ty.FriendlyName())
			}
			for name, aty := range ty.AttributeTypes() {
				atys[name] = aty
			}
			continue
		}
		if keyValStr, err := convert.Convert(keyVal, cty.String); err == nil {
			keyVal = keyValStr
		} else {
			return cty.NilType, parseEventErrorf(&nextEvt, "only strings are allowed as mapping keys")
		}
		if keyVal.IsNull() {
			return cty.NilType, parseEventErrorf(&nextEvt, "mapping key cannot be null")
		}
		if !keyVal.IsKnown() {
			return cty.NilType, parseEventErrorf(&nextEvt, "mapping key must be known")
		}
		valTy, err := c.impliedTypeParse(an, p)
		if err != nil {
			return cty.NilType, err
		}

		atys[keyVal.AsString()] = valTy
	}
}

func (c *Converter) impliedTypeSequence(an *typeAnalysis, evt *yaml_event_t, p *yaml_parser_t) (cty.Type, error) {
	tag := string(evt.tag)
	anchor := string(evt.anchor)

	if tag != "" && tag != yaml_SEQ_TAG {
		return cty.NilType, parseEventErrorf(evt, "can't interpret sequence as %s", tag)
	}

	if anchor != "" {
		an.beginAnchor(anchor)
	}

	var atys []cty.Type
	for {
		var nextEvt yaml_event_t
		if !yaml_parser_parse(p, &nextEvt) {
			return cty.NilType, parserError(p)
		}
		if nextEvt.typ == yaml_SEQUENCE_END_EVENT {
			ty := cty.Tuple(atys)
			if anchor != "" {
				an.completeAnchor(anchor, ty)
			}
			return ty, nil
		}

		valTy, err := c.impliedTypeParseRemainder(an, &nextEvt, p)
		if err != nil {
			return cty.NilType, err
		}

		atys = append(atys, valTy)
	}
}

func (c *Converter) impliedTypeAlias(an *typeAnalysis, evt *yaml_event_t, p *yaml_parser_t) (cty.Type, error) {
	ty, err := an.anchorType(string(evt.anchor))
	if err != nil {
		err = parseEventErrorWrap(evt, err)
	}
	return ty, err
}

type typeAnalysis struct {
	anchorsPending map[string]int
	anchorTypes    map[string]cty.Type
}

func (an *typeAnalysis) beginAnchor(name string) {
	an.anchorsPending[name]++
}

func (an *typeAnalysis) completeAnchor(name string, ty cty.Type) {
	an.anchorsPending[name]--
	if an.anchorsPending[name] == 0 {
		delete(an.anchorsPending, name)
	}
	an.anchorTypes[name] = ty
}

func (an *typeAnalysis) anchorType(name string) (cty.Type, error) {
	if _, pending := an.anchorsPending[name]; pending {
		// YAML normally allows self-referencing structures, but cty cannot
		// represent them (it requires all structures to be finite) so we
		// must fail here.
		return cty.NilType, fmt.Errorf("cannot refer to anchor %q from inside its own definition", name)
	}
	ty, ok := an.anchorTypes[name]
	if !ok {
		return cty.NilType, fmt.Errorf("reference to undefined anchor %q", name)
	}
	return ty, nil
}
