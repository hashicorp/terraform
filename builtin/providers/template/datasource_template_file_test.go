package template

import (
	"fmt"
	"strings"
	"sync"
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
		{`{a="foo"}`, `$${a}`, `foo`},
		{`{a="hello"}`, `$${replace(a, "ello", "i")}`, `hi`},
		{`{}`, `${1+2+3}`, `6`},
		{`{}`, `/`, `/`},
	}

	for _, tt := range cases {
		r.UnitTest(t, r.TestCase{
			Providers: testProviders,
			Steps: []r.TestStep{
				r.TestStep{
					Config: testTemplateConfig(tt.template, tt.vars),
					Check: func(s *terraform.State) error {
						got := s.RootModule().Outputs["rendered"]
						if tt.want != got.Value {
							return fmt.Errorf("template:\n%s\nvars:\n%s\ngot:\n%s\nwant:\n%s\n", tt.template, tt.vars, got, tt.want)
						}
						return nil
					},
				},
			},
		})
	}
}

func TestValidateVarsAttribute(t *testing.T) {
	cases := map[string]struct {
		Vars      map[string]interface{}
		ExpectErr string
	}{
		"lists are invalid": {
			map[string]interface{}{
				"list": []interface{}{},
			},
			`vars: cannot contain non-primitives`,
		},
		"maps are invalid": {
			map[string]interface{}{
				"map": map[string]interface{}{},
			},
			`vars: cannot contain non-primitives`,
		},
		"strings, integers, floats, and bools are AOK": {
			map[string]interface{}{
				"string": "foo",
				"int":    1,
				"bool":   true,
				"float":  float64(1.0),
			},
			``,
		},
	}

	for tn, tc := range cases {
		_, es := validateVarsAttribute(tc.Vars, "vars")
		if len(es) > 0 {
			if tc.ExpectErr == "" {
				t.Fatalf("%s: expected no err, got: %#v", tn, es)
			}
			if !strings.Contains(es[0].Error(), tc.ExpectErr) {
				t.Fatalf("%s: expected\n%s\nto contain\n%s", tn, es[0], tc.ExpectErr)
			}
		} else if tc.ExpectErr != "" {
			t.Fatalf("%s: expected err containing %q, got none!", tn, tc.ExpectErr)
		}
	}
}

// This test covers a panic due to config.Func formerly being a
// shared map, causing multiple template_file resources to try and
// accessing it parallel during their lang.Eval() runs.
//
// Before fix, test fails under `go test -race`
func TestTemplateSharedMemoryRace(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(t *testing.T, i int) {
			out, err := execute("don't panic!", map[string]interface{}{})
			if err != nil {
				t.Fatalf("err: %s", err)
			}
			if out != "don't panic!" {
				t.Fatalf("bad output: %s", out)
			}
			wg.Done()
		}(t, i)
	}
	wg.Wait()
}

func testTemplateConfig(template, vars string) string {
	return fmt.Sprintf(`
		data "template_file" "t0" {
			template = "%s"
			vars = %s
		}
		output "rendered" {
				value = "${data.template_file.t0.rendered}"
		}`, template, vars)
}
