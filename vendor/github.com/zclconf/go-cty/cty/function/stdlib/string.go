package stdlib

import (
	"strings"

	"github.com/apparentlymart/go-textseg/textseg"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/gocty"
)

var UpperFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "str",
			Type:             cty.String,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		in := args[0].AsString()
		out := strings.ToUpper(in)
		return cty.StringVal(out), nil
	},
})

var LowerFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "str",
			Type:             cty.String,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		in := args[0].AsString()
		out := strings.ToLower(in)
		return cty.StringVal(out), nil
	},
})

var ReverseFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "str",
			Type:             cty.String,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		in := []byte(args[0].AsString())
		out := make([]byte, len(in))
		pos := len(out)

		inB := []byte(in)
		for i := 0; i < len(in); {
			d, _, _ := textseg.ScanGraphemeClusters(inB[i:], true)
			cluster := in[i : i+d]
			pos -= len(cluster)
			copy(out[pos:], cluster)
			i += d
		}

		return cty.StringVal(string(out)), nil
	},
})

var StrlenFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "str",
			Type:             cty.String,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		in := args[0].AsString()
		l := 0

		inB := []byte(in)
		for i := 0; i < len(in); {
			d, _, _ := textseg.ScanGraphemeClusters(inB[i:], true)
			l++
			i += d
		}

		return cty.NumberIntVal(int64(l)), nil
	},
})

var SubstrFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "str",
			Type:             cty.String,
			AllowDynamicType: true,
		},
		{
			Name:             "offset",
			Type:             cty.Number,
			AllowDynamicType: true,
		},
		{
			Name:             "length",
			Type:             cty.Number,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		in := []byte(args[0].AsString())
		var offset, length int

		var err error
		err = gocty.FromCtyValue(args[1], &offset)
		if err != nil {
			return cty.NilVal, err
		}
		err = gocty.FromCtyValue(args[2], &length)
		if err != nil {
			return cty.NilVal, err
		}

		if offset < 0 {
			totalLenNum, err := Strlen(args[0])
			if err != nil {
				// should never happen
				panic("Stdlen returned an error")
			}

			var totalLen int
			err = gocty.FromCtyValue(totalLenNum, &totalLen)
			if err != nil {
				// should never happen
				panic("Stdlen returned a non-int number")
			}

			offset += totalLen
		}

		sub := in
		pos := 0
		var i int

		// First we'll seek forward to our offset
		if offset > 0 {
			for i = 0; i < len(sub); {
				d, _, _ := textseg.ScanGraphemeClusters(sub[i:], true)
				i += d
				pos++
				if pos == offset {
					break
				}
				if i >= len(in) {
					return cty.StringVal(""), nil
				}
			}

			sub = sub[i:]
		}

		if length < 0 {
			// Taking the remainder of the string is a fast path since
			// we can just return the rest of the buffer verbatim.
			return cty.StringVal(string(sub)), nil
		}

		// Otherwise we need to start seeking forward again until we
		// reach the length we want.
		pos = 0
		for i = 0; i < len(sub); {
			d, _, _ := textseg.ScanGraphemeClusters(sub[i:], true)
			i += d
			pos++
			if pos == length {
				break
			}
		}

		sub = sub[:i]

		return cty.StringVal(string(sub)), nil
	},
})

// Upper is a Function that converts a given string to uppercase.
func Upper(str cty.Value) (cty.Value, error) {
	return UpperFunc.Call([]cty.Value{str})
}

// Lower is a Function that converts a given string to lowercase.
func Lower(str cty.Value) (cty.Value, error) {
	return LowerFunc.Call([]cty.Value{str})
}

// Reverse is a Function that reverses the order of the characters in the
// given string.
//
// As usual, "character" for the sake of this function is a grapheme cluster,
// so combining diacritics (for example) will be considered together as a
// single character.
func Reverse(str cty.Value) (cty.Value, error) {
	return ReverseFunc.Call([]cty.Value{str})
}

// Strlen is a Function that returns the length of the given string in
// characters.
//
// As usual, "character" for the sake of this function is a grapheme cluster,
// so combining diacritics (for example) will be considered together as a
// single character.
func Strlen(str cty.Value) (cty.Value, error) {
	return StrlenFunc.Call([]cty.Value{str})
}

// Substr is a Function that extracts a sequence of characters from another
// string and creates a new string.
//
// As usual, "character" for the sake of this function is a grapheme cluster,
// so combining diacritics (for example) will be considered together as a
// single character.
//
// The "offset" index may be negative, in which case it is relative to the
// end of the given string.
//
// The "length" may be -1, in which case the remainder of the string after
// the given offset will be returned.
func Substr(str cty.Value, offset cty.Value, length cty.Value) (cty.Value, error) {
	return SubstrFunc.Call([]cty.Value{str, offset, length})
}
