package command

import (
	"path"
	"strings"
	"testing"
)

func TestTest(t *testing.T) {
	tcs := map[string]struct {
		args     []string
		expected string
		code     int
	}{
		"simple_pass": {
			expected: "1 passed, 0 failed.",
			code:     0,
		},
		"simple_pass_nested": {
			expected: "1 passed, 0 failed.",
			code:     0,
		},
		"pass_with_locals": {
			expected: "1 passed, 0 failed.",
			code:     0,
		},
		"pass_with_variables": {
			expected: "2 passed, 0 failed.",
			code:     0,
		},
		"plan_then_apply": {
			expected: "2 passed, 0 failed.",
			code:     0,
		},
		"simple_fail": {
			expected: "0 passed, 1 failed.",
			code:     1,
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			td := t.TempDir()
			testCopyDir(t, testFixturePath(path.Join("test", name)), td)
			defer testChdir(t, td)()

			p := planFixtureProvider()
			view, done := testView(t)

			c := &TestCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(p),
					View:             view,
				},
			}

			code := c.Run(tc.args)
			output := done(t)

			if code != tc.code {
				t.Errorf("expected status code %d but got %d", tc.code, code)
			}

			if !strings.Contains(output.Stdout(), tc.expected) {
				t.Errorf("output didn't contain expected string:\n\n%s", output.All())
			}
		})
	}
}
