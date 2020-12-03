package funcs

import (
	"fmt"
	"net/url"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// URLParseFunc constructs a function that parses a url in its parts: [scheme:][//[userinfo@]host:port][/]path[?query][#fragment]
var URLParseFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "url",
			Type: cty.String,
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		if args[0].IsNull() {
			return cty.NilType, function.NewArgErrorf(0, "URL cannot be empty")
		}
		return cty.DynamicPseudoType, nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		urlStr := args[0].AsString()

		u, err := url.Parse(urlStr)
		if err != nil {
			return cty.NilVal, fmt.Errorf("Failed to parse url '%s' error: %v", urlStr, err)
		}

		pw, pwSet := u.User.Password()
		result := map[string]cty.Value{
			"scheme": cty.StringVal(u.Scheme),
			"user": cty.ObjectVal(map[string]cty.Value{
				"username":     cty.StringVal(u.User.Username()),
				"password":     cty.StringVal(pw),
				"password_set": cty.BoolVal(pwSet),
			}),
			"host":     cty.StringVal(u.Host),
			"hostname": cty.StringVal(u.Hostname()),
			"port":     cty.StringVal(u.Port()),
			"path":     cty.StringVal(u.Path),
			"fragment": cty.StringVal(u.Fragment),
		}
		q := u.Query()
		if len(q) > 0 {
			query := make(map[string]cty.Value)
			for k, v := range q {
				vals := make([]cty.Value, len(v))
				for i := range v {
					vals[i] = cty.StringVal(v[i])
				}
				query[k] = cty.ListVal(vals)
			}
			result["query"] = cty.MapVal(query)
		}
		return cty.ObjectVal(result), nil
	},
})

// URLParse searches a given string for another
// given substring and return the index of the start of the first ocurrence.
func URLParse(url cty.Value) (cty.Value, error) {
	return URLParseFunc.Call([]cty.Value{url})
}
