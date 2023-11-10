package funcs

import (
	u "net/url"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// ParseURLFunc takes a URL string and returns map[string]string
// containing the URL's components https://www.rfc-editor.org/rfc/rfc3986#appendix-B
var ParseURLFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "url",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.Map(cty.String)),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {

		str := args[0].AsString()

		url, err := u.Parse(str)

		if err != nil {
			return cty.Value{}, err
		}

		outMap := make(map[string]cty.Value)

		password, _ := url.User.Password()

		outMap["Password"] = cty.StringVal(password)
		outMap["Username"] = cty.StringVal(url.User.Username())
		outMap["Hostname"] = cty.StringVal(url.Hostname())
		outMap["Port"] = cty.StringVal(url.Port())
		outMap["Fragment"] = cty.StringVal(url.Fragment)
		outMap["Path"] = cty.StringVal(url.Path)
		outMap["Scheme"] = cty.StringVal(url.Scheme)
		outMap["RawQuery"] = cty.StringVal(url.RawQuery)

		return cty.MapVal(outMap), nil
	},
})
