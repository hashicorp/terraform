package repl

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/zclconf/go-cty/cty"
)

func TestFormatValue(t *testing.T) {
	tests := []struct {
		Val  cty.Value
		Want string
	}{
		{
			cty.NullVal(cty.DynamicPseudoType),
			`null`,
		},
		{
			cty.NullVal(cty.String),
			`tostring(null)`,
		},
		{
			cty.NullVal(cty.Number),
			`tonumber(null)`,
		},
		{
			cty.NullVal(cty.Bool),
			`tobool(null)`,
		},
		{
			cty.NullVal(cty.List(cty.String)),
			`tolist(null) /* of string */`,
		},
		{
			cty.NullVal(cty.Set(cty.Number)),
			`toset(null) /* of number */`,
		},
		{
			cty.NullVal(cty.Map(cty.Bool)),
			`tomap(null) /* of bool */`,
		},
		{
			cty.NullVal(cty.Object(map[string]cty.Type{"a": cty.Bool})),
			`null /* object */`, // Ideally this would display the full object type, including its attributes
		},
		{
			cty.UnknownVal(cty.DynamicPseudoType),
			`(known after apply)`,
		},
		{
			cty.StringVal(""),
			`""`,
		},
		{
			cty.StringVal("hello"),
			`"hello"`,
		},
		{
			cty.StringVal("hello\nworld"),
			`<<EOT
hello
world
EOT`,
		},
		{
			cty.StringVal("EOR\nEOS\nEOT\nEOU"),
			`<<EOT_
EOR
EOS
EOT
EOU
EOT_`,
		},
		{
			cty.ObjectVal(map[string]cty.Value{"foo": cty.StringVal("boop\nbeep")}),
			`{
  "foo" = <<-EOT
  boop
  beep
  EOT
}`,
		},
		{
			cty.Zero,
			`0`,
		},
		{
			cty.NumberIntVal(5),
			`5`,
		},
		{
			cty.NumberIntVal(1234567890),
			`1234567890`,
		},
		{
			cty.NumberFloatVal(5.2),
			`5.2`,
		},
		{
			cty.NumberFloatVal(123456789.0),
			`123456789`,
		},
		{
			cty.NumberFloatVal(123456789.01),
			`123456789.01`,
		},
		{
			cty.False,
			`false`,
		},
		{
			cty.True,
			`true`,
		},
		{
			cty.EmptyObjectVal,
			`{}`,
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("b"),
			}),
			`{
  "a" = "b"
}`,
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("b"),
				"c": cty.StringVal("d"),
			}),
			`{
  "a" = "b"
  "c" = "d"
}`,
		},
		{
			cty.MapValEmpty(cty.String),
			`tomap({})`,
		},
		{
			cty.EmptyTupleVal,
			`[]`,
		},
		{
			cty.TupleVal([]cty.Value{
				cty.StringVal("b"),
			}),
			`[
  "b",
]`,
		},
		{
			cty.TupleVal([]cty.Value{
				cty.StringVal("b"),
				cty.StringVal("d"),
			}),
			`[
  "b",
  "d",
]`,
		},
		{
			cty.ListValEmpty(cty.String),
			`tolist([])`,
		},
		{
			cty.SetValEmpty(cty.String),
			`toset([])`,
		},
		{
			cty.StringVal("sensitive value").Mark(marks.Sensitive),
			"(sensitive)",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v", test.Val), func(t *testing.T) {
			got := FormatValue(test.Val, 0)
			if got != test.Want {
				t.Errorf("wrong result\nvalue: %#v\ngot:   %s\nwant:  %s", test.Val, got, test.Want)
			}
		})
	}
}
