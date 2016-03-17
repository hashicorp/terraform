package template

import (
	"fmt"
	"io/ioutil"
	"os"
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
		{`{a="foo"}`, `${a}`, `foo`},
		{`{a="hello"}`, `${replace(a, "ello", "i")}`, `hi`},
		{`{}`, `${1+2+3}`, `6`},
	}

	for _, tt := range cases {
		r.Test(t, r.TestCase{
			Providers: testProviders,
			Steps: []r.TestStep{
				r.TestStep{
					Config: testTemplateConfig(tt.template, tt.vars),
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

// https://github.com/hashicorp/terraform/issues/2344
func TestTemplateVariableChange(t *testing.T) {
	steps := []struct {
		vars     string
		template string
		want     string
	}{
		{`{a="foo"}`, `${a}`, `foo`},
		{`{b="bar"}`, `${b}`, `bar`},
	}

	var testSteps []r.TestStep
	for i, step := range steps {
		testSteps = append(testSteps, r.TestStep{
			Config: testTemplateConfig(step.template, step.vars),
			Check: func(i int, want string) r.TestCheckFunc {
				return func(s *terraform.State) error {
					got := s.RootModule().Outputs["rendered"]
					if want != got {
						return fmt.Errorf("[%d] got:\n%q\nwant:\n%q\n", i, got, want)
					}
					return nil
				}
			}(i, step.want),
		})
	}

	r.Test(t, r.TestCase{
		Providers: testProviders,
		Steps:     testSteps,
	})
}

func TestValidateTemplateAttribute(t *testing.T) {
	file, err := ioutil.TempFile("", "testtemplate")
	if err != nil {
		t.Fatal(err)
	}
	file.WriteString("Hello world.")
	file.Close()
	defer os.Remove(file.Name())

	ws, es := validateTemplateAttribute(file.Name(), "test")

	if len(es) != 0 {
		t.Fatalf("Unexpected errors: %#v", es)
	}

	if len(ws) != 1 {
		t.Fatalf("Expected 1 warning, got %d", len(ws))
	}

	if !strings.Contains(ws[0], "Specifying a path directly is deprecated") {
		t.Fatalf("Expected warning about path, got: %s", ws[0])
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
		go func(wg sync.WaitGroup, t *testing.T, i int) {
			wg.Add(1)
			out, err := execute("don't panic!", map[string]interface{}{})
			if err != nil {
				t.Fatalf("err: %s", err)
			}
			if out != "don't panic!" {
				t.Fatalf("bad output: %s", out)
			}
			wg.Done()
		}(wg, t, i)
	}
	wg.Wait()
}

func testTemplateConfig(template, vars string) string {
	return fmt.Sprintf(`
		resource "template_file" "t0" {
			template = "%s"
			vars = %s
		}
		output "rendered" {
				value = "${template_file.t0.rendered}"
		}`, template, vars)
}
