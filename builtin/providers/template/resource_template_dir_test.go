package template

import (
	"fmt"
	"testing"

	"errors"
	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"io/ioutil"
	"os"
	"path/filepath"
)

const templateDirRenderingConfig = `
resource "template_dir" "dir" {
	source_dir = "%s"
  destination_dir = "%s"
  vars = %s
}`

type testTemplate struct {
	template string
	want     string
}

func testTemplateDirWriteFiles(files map[string]testTemplate) (in, out string, err error) {
	in, err = ioutil.TempDir(os.TempDir(), "terraform_template_dir")
	if err != nil {
		return
	}

	for name, file := range files {
		path := filepath.Join(in, name)

		err = os.MkdirAll(filepath.Dir(path), 0777)
		if err != nil {
			return
		}

		err = ioutil.WriteFile(path, []byte(file.template), 0777)
		if err != nil {
			return
		}
	}

	out = fmt.Sprintf("%s.out", in)
	return
}

func TestTemplateDirRendering(t *testing.T) {
	var cases = []struct {
		vars  string
		files map[string]testTemplate
	}{
		{
			files: map[string]testTemplate{
				"foo.txt":           {"${bar}", "bar"},
				"nested/monkey.txt": {"ooh-ooh-ooh-eee-eee", "ooh-ooh-ooh-eee-eee"},
				"maths.txt":         {"${1+2+3}", "6"},
			},
			vars: `{bar = "bar"}`,
		},
	}

	for _, tt := range cases {
		// Write the desired templates in a temporary directory.
		in, out, err := testTemplateDirWriteFiles(tt.files)
		if err != nil {
			t.Skipf("could not write templates to temporary directory: %s", err)
			continue
		}
		defer os.RemoveAll(in)
		defer os.RemoveAll(out)

		// Run test case.
		r.UnitTest(t, r.TestCase{
			Providers: testProviders,
			Steps: []r.TestStep{
				{
					Config: fmt.Sprintf(templateDirRenderingConfig, in, out, tt.vars),
					Check: func(s *terraform.State) error {
						for name, file := range tt.files {
							content, err := ioutil.ReadFile(filepath.Join(out, name))
							if err != nil {
								return fmt.Errorf("template:\n%s\nvars:\n%s\ngot:\n%s\nwant:\n%s\n", file.template, tt.vars, err, file.want)
							}
							if string(content) != file.want {
								return fmt.Errorf("template:\n%s\nvars:\n%s\ngot:\n%s\nwant:\n%s\n", file.template, tt.vars, content, file.want)
							}
						}
						return nil
					},
				},
			},
			CheckDestroy: func(*terraform.State) error {
				if _, err := os.Stat(out); os.IsNotExist(err) {
					return nil
				}
				return errors.New("template_dir did not get destroyed")
			},
		})
	}
}
