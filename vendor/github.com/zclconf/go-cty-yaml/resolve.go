package yaml

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/zclconf/go-cty/cty"
)

type resolveMapItem struct {
	value cty.Value
	tag   string
}

var resolveTable = make([]byte, 256)
var resolveMap = make(map[string]resolveMapItem)

func init() {
	t := resolveTable
	t[int('+')] = 'S' // Sign
	t[int('-')] = 'S'
	for _, c := range "0123456789" {
		t[int(c)] = 'D' // Digit
	}
	for _, c := range "yYnNtTfFoO~" {
		t[int(c)] = 'M' // In map
	}
	t[int('.')] = '.' // Float (potentially in map)

	var resolveMapList = []struct {
		v   cty.Value
		tag string
		l   []string
	}{
		{cty.True, yaml_BOOL_TAG, []string{"y", "Y", "yes", "Yes", "YES"}},
		{cty.True, yaml_BOOL_TAG, []string{"true", "True", "TRUE"}},
		{cty.True, yaml_BOOL_TAG, []string{"on", "On", "ON"}},
		{cty.False, yaml_BOOL_TAG, []string{"n", "N", "no", "No", "NO"}},
		{cty.False, yaml_BOOL_TAG, []string{"false", "False", "FALSE"}},
		{cty.False, yaml_BOOL_TAG, []string{"off", "Off", "OFF"}},
		{cty.NullVal(cty.DynamicPseudoType), yaml_NULL_TAG, []string{"", "~", "null", "Null", "NULL"}},
		{cty.PositiveInfinity, yaml_FLOAT_TAG, []string{".inf", ".Inf", ".INF"}},
		{cty.PositiveInfinity, yaml_FLOAT_TAG, []string{"+.inf", "+.Inf", "+.INF"}},
		{cty.NegativeInfinity, yaml_FLOAT_TAG, []string{"-.inf", "-.Inf", "-.INF"}},
	}

	m := resolveMap
	for _, item := range resolveMapList {
		for _, s := range item.l {
			m[s] = resolveMapItem{item.v, item.tag}
		}
	}
}

const longTagPrefix = "tag:yaml.org,2002:"

func shortTag(tag string) string {
	// TODO This can easily be made faster and produce less garbage.
	if strings.HasPrefix(tag, longTagPrefix) {
		return "!!" + tag[len(longTagPrefix):]
	}
	return tag
}

func longTag(tag string) string {
	if strings.HasPrefix(tag, "!!") {
		return longTagPrefix + tag[2:]
	}
	return tag
}

func resolvableTag(tag string) bool {
	switch tag {
	case "", yaml_STR_TAG, yaml_BOOL_TAG, yaml_INT_TAG, yaml_FLOAT_TAG, yaml_NULL_TAG, yaml_TIMESTAMP_TAG, yaml_BINARY_TAG:
		return true
	}
	return false
}

var yamlStyleFloat = regexp.MustCompile(`^[-+]?(\.[0-9]+|[0-9]+(\.[0-9]*)?)([eE][-+]?[0-9]+)?$`)

func (c *Converter) resolveScalar(tag string, src string, style yaml_scalar_style_t) (cty.Value, error) {
	if !resolvableTag(tag) {
		return cty.NilVal, fmt.Errorf("unsupported tag %q", tag)
	}

	// Any data is accepted as a !!str or !!binary.
	// Otherwise, the prefix is enough of a hint about what it might be.
	hint := byte('N')
	if src != "" {
		hint = resolveTable[src[0]]
	}
	if hint != 0 && tag != yaml_STR_TAG && tag != yaml_BINARY_TAG {
		if style == yaml_SINGLE_QUOTED_SCALAR_STYLE || style == yaml_DOUBLE_QUOTED_SCALAR_STYLE {
			return cty.StringVal(src), nil
		}

		// Handle things we can lookup in a map.
		if item, ok := resolveMap[src]; ok {
			return item.value, nil
		}

		if tag == "" {
			for _, nan := range []string{".nan", ".NaN", ".NAN"} {
				if src == nan {
					// cty cannot represent NaN, so this is an error
					return cty.NilVal, fmt.Errorf("floating point NaN is not supported")
				}
			}
		}

		// Base 60 floats are intentionally not supported.

		switch hint {
		case 'M':
			// We've already checked the map above.

		case '.':
			// Not in the map, so maybe a normal float.
			if numberVal, err := cty.ParseNumberVal(src); err == nil {
				return numberVal, nil
			}

		case 'D', 'S':
			// Int, float, or timestamp.
			// Only try values as a timestamp if the value is unquoted or there's an explicit
			// !!timestamp tag.
			if tag == "" || tag == yaml_TIMESTAMP_TAG {
				t, ok := parseTimestamp(src)
				if ok {
					// cty has no timestamp type, but its functions stdlib
					// conventionally uses strings in an RFC3339 encoding
					// to represent time, so we'll follow that convention here.
					return cty.StringVal(t.Format(time.RFC3339)), nil
				}
			}

			plain := strings.Replace(src, "_", "", -1)
			if numberVal, err := cty.ParseNumberVal(plain); err == nil {
				return numberVal, nil
			}
			if strings.HasPrefix(plain, "0b") || strings.HasPrefix(plain, "-0b") {
				tag = yaml_INT_TAG // will handle parsing below in our tag switch
			}
		default:
			panic(fmt.Sprintf("cannot resolve tag %q with source %q", tag, src))
		}
	}

	if tag == "" && src == "<<" {
		return mergeMappingVal, nil
	}

	switch tag {
	case yaml_STR_TAG, yaml_BINARY_TAG:
		// If it's binary then we want to keep the base64 representation, because
		// cty has no binary type, but we will check that it's actually base64.
		if tag == yaml_BINARY_TAG {
			_, err := base64.StdEncoding.DecodeString(src)
			if err != nil {
				return cty.NilVal, fmt.Errorf("cannot parse %q as %s: not valid base64", src, tag)
			}
		}
		return cty.StringVal(src), nil
	case yaml_BOOL_TAG:
		item, ok := resolveMap[src]
		if !ok || item.tag != yaml_BOOL_TAG {
			return cty.NilVal, fmt.Errorf("cannot parse %q as %s", src, tag)
		}
		return item.value, nil
	case yaml_FLOAT_TAG, yaml_INT_TAG:
		// Note: We don't actually check that a value tagged INT is a whole
		// number here. We could, but cty generally doesn't care about the
		// int/float distinction, so we'll just be generous and accept it.
		plain := strings.Replace(src, "_", "", -1)
		if numberVal, err := cty.ParseNumberVal(plain); err == nil { // handles decimal integers and floats
			return numberVal, nil
		}
		if intv, err := strconv.ParseInt(plain, 0, 64); err == nil { // handles 0x and 00 prefixes
			return cty.NumberIntVal(intv), nil
		}
		if uintv, err := strconv.ParseUint(plain, 0, 64); err == nil { // handles 0x and 00 prefixes
			return cty.NumberUIntVal(uintv), nil
		}
		if strings.HasPrefix(plain, "0b") {
			intv, err := strconv.ParseInt(plain[2:], 2, 64)
			if err == nil {
				return cty.NumberIntVal(intv), nil
			}
			uintv, err := strconv.ParseUint(plain[2:], 2, 64)
			if err == nil {
				return cty.NumberUIntVal(uintv), nil
			}
		} else if strings.HasPrefix(plain, "-0b") {
			intv, err := strconv.ParseInt("-"+plain[3:], 2, 64)
			if err == nil {
				return cty.NumberIntVal(intv), nil
			}
		}
		return cty.NilVal, fmt.Errorf("cannot parse %q as %s", src, tag)
	case yaml_TIMESTAMP_TAG:
		t, ok := parseTimestamp(src)
		if ok {
			// cty has no timestamp type, but its functions stdlib
			// conventionally uses strings in an RFC3339 encoding
			// to represent time, so we'll follow that convention here.
			return cty.StringVal(t.Format(time.RFC3339)), nil
		}
		return cty.NilVal, fmt.Errorf("cannot parse %q as %s", src, tag)
	case yaml_NULL_TAG:
		return cty.NullVal(cty.DynamicPseudoType), nil
	case "":
		return cty.StringVal(src), nil
	default:
		return cty.NilVal, fmt.Errorf("unsupported tag %q", tag)
	}
}

// encodeBase64 encodes s as base64 that is broken up into multiple lines
// as appropriate for the resulting length.
func encodeBase64(s string) string {
	const lineLen = 70
	encLen := base64.StdEncoding.EncodedLen(len(s))
	lines := encLen/lineLen + 1
	buf := make([]byte, encLen*2+lines)
	in := buf[0:encLen]
	out := buf[encLen:]
	base64.StdEncoding.Encode(in, []byte(s))
	k := 0
	for i := 0; i < len(in); i += lineLen {
		j := i + lineLen
		if j > len(in) {
			j = len(in)
		}
		k += copy(out[k:], in[i:j])
		if lines > 1 {
			out[k] = '\n'
			k++
		}
	}
	return string(out[:k])
}

// This is a subset of the formats allowed by the regular expression
// defined at http://yaml.org/type/timestamp.html.
var allowedTimestampFormats = []string{
	"2006-1-2T15:4:5.999999999Z07:00", // RCF3339Nano with short date fields.
	"2006-1-2t15:4:5.999999999Z07:00", // RFC3339Nano with short date fields and lower-case "t".
	"2006-1-2 15:4:5.999999999",       // space separated with no time zone
	"2006-1-2",                        // date only
	// Notable exception: time.Parse cannot handle: "2001-12-14 21:59:43.10 -5"
	// from the set of examples.
}

// parseTimestamp parses s as a timestamp string and
// returns the timestamp and reports whether it succeeded.
// Timestamp formats are defined at http://yaml.org/type/timestamp.html
func parseTimestamp(s string) (time.Time, bool) {
	// TODO write code to check all the formats supported by
	// http://yaml.org/type/timestamp.html instead of using time.Parse.

	// Quick check: all date formats start with YYYY-.
	i := 0
	for ; i < len(s); i++ {
		if c := s[i]; c < '0' || c > '9' {
			break
		}
	}
	if i != 4 || i == len(s) || s[i] != '-' {
		return time.Time{}, false
	}
	for _, format := range allowedTimestampFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

type mergeMapping struct{}

var mergeMappingTy = cty.Capsule("merge mapping", reflect.TypeOf(mergeMapping{}))
var mergeMappingVal = cty.CapsuleVal(mergeMappingTy, &mergeMapping{})
