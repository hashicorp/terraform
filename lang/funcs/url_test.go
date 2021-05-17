package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestURLParse(t *testing.T) {
	tests := []struct {
		URL  cty.Value
		Want cty.Value
		Err  bool
	}{
		{ // valid url
			cty.StringVal("mongodb://username:password@hostN:123/path/?query=query#fragment"),
			cty.ObjectVal(map[string]cty.Value{
				"scheme": cty.StringVal("mongodb"),
				"user": cty.ObjectVal(map[string]cty.Value{
					"username":     cty.StringVal("username"),
					"password":     cty.StringVal("password"),
					"password_set": cty.BoolVal(true),
				}),
				"host":     cty.StringVal("hostN:123"),
				"hostname": cty.StringVal("hostN"),
				"port":     cty.StringVal("123"),
				"path":     cty.StringVal("/path/"),
				"fragment": cty.StringVal("fragment"),
				"query": cty.MapVal(map[string]cty.Value{
					"query": cty.ListVal([]cty.Value{cty.StringVal("query")}),
				}),
			}),
			false,
		},
		{ // valid url no query
			cty.StringVal("http://hostname/path/"),
			cty.ObjectVal(map[string]cty.Value{
				"scheme": cty.StringVal("http"),
				"user": cty.ObjectVal(map[string]cty.Value{
					"username":     cty.StringVal(""),
					"password":     cty.StringVal(""),
					"password_set": cty.BoolVal(false),
				}),
				"host":     cty.StringVal("hostname"),
				"hostname": cty.StringVal("hostname"),
				"port":     cty.StringVal(""),
				"path":     cty.StringVal("/path/"),
				"fragment": cty.StringVal(""),
			}),
			false,
		},
		{ // invalid url
			cty.StringVal("https://test:port"),
			cty.NilVal,
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("url_decode(%#v)", test.URL), func(t *testing.T) {
			got, err := URLParse(test.URL)

			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}
