package template

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

var testProviders = map[string]terraform.ResourceProvider{
	"template": Provider(),
}

func TestTemplateRendering(t *testing.T) {
	var cases = []struct {
		vars     string
		template string
		want     string
	}{
		{`{}`, `ABC`, `ABC`},
		{`{a="foo"}`, `${a}`, `foo`},
		{`{a="hello"}`, `${replace(a, "ello", "i")}`, `hi`},
		{`{}`, `${1+2+3}`, `6`},
	}

	for _, tt := range cases {
		r.Test(t, r.TestCase{
			PreCheck: func() {
				readfile = func(string) ([]byte, error) {
					return []byte(tt.template), nil
				}
			},
			Providers: testProviders,
			Steps: []r.TestStep{
				r.TestStep{
					Config: `
resource "template_file" "t0" {
	filename = "mock"
	vars = ` + tt.vars + `
}
output "rendered" {
    value = "${template_file.t0.rendered}"
}
`,
					Check: func(s *terraform.State) error {
						got := s.RootModule().Outputs["rendered"]
						if tt.want != got {
							return fmt.Errorf("template:\n%s\nvars:\n%s\ngot:\n%s\nwant:\n%s\n", tt.template, tt.vars, got, tt.want)
						}
						return nil
					},
				},
			},
		})
	}
}
