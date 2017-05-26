package terraform

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/config"
)

func TestEvalValidateResourceSelfRef(t *testing.T) {
	cases := []struct {
		Name   string
		Addr   string
		Config map[string]interface{}
		Err    bool
	}{
		{
			"no interpolations",
			"aws_instance.foo",
			map[string]interface{}{
				"foo": "bar",
			},
			false,
		},

		{
			"non self reference",
			"aws_instance.foo",
			map[string]interface{}{
				"foo": "${aws_instance.bar.id}",
			},
			false,
		},

		{
			"self reference",
			"aws_instance.foo",
			map[string]interface{}{
				"foo": "hello ${aws_instance.foo.id}",
			},
			true,
		},

		{
			"self reference other index",
			"aws_instance.foo",
			map[string]interface{}{
				"foo": "hello ${aws_instance.foo.4.id}",
			},
			false,
		},

		{
			"self reference same index",
			"aws_instance.foo[4]",
			map[string]interface{}{
				"foo": "hello ${aws_instance.foo.4.id}",
			},
			true,
		},

		{
			"self reference multi",
			"aws_instance.foo[4]",
			map[string]interface{}{
				"foo": "hello ${aws_instance.foo.*.id}",
			},
			true,
		},

		{
			"self reference multi single",
			"aws_instance.foo",
			map[string]interface{}{
				"foo": "hello ${aws_instance.foo.*.id}",
			},
			true,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			addr, err := ParseResourceAddress(tc.Addr)
			if err != nil {
				t.Fatalf("err: %s", err)
			}
			conf := config.TestRawConfig(t, tc.Config)

			n := &EvalValidateResourceSelfRef{Addr: &addr, Config: &conf}
			result, err := n.Eval(nil)
			if result != nil {
				t.Fatal("result should always be nil")
			}
			if (err != nil) != tc.Err {
				t.Fatalf("err: %s", err)
			}
		})
	}
}
