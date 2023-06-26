package views

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestTestHuman_Conclusion(t *testing.T) {
	tcs := map[string]struct {
		Suite    *moduletest.Suite
		Expected string
	}{
		"no tests": {
			Suite:    &moduletest.Suite{},
			Expected: "\nExecuted 0 tests.\n",
		},

		"only skipped tests": {
			Suite: &moduletest.Suite{
				Status: moduletest.Skip,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest": {
						Name:   "descriptive_test_name.tftest",
						Status: moduletest.Skip,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_two",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_three",
								Status: moduletest.Skip,
							},
						},
					},
					"other_descriptive_test_name.tftest": {
						Name:   "other_descriptive_test_name.tftest",
						Status: moduletest.Skip,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_two",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_three",
								Status: moduletest.Skip,
							},
						},
					},
				},
			},
			Expected: "\nExecuted 0 tests, 6 skipped.\n",
		},

		"only passed tests": {
			Suite: &moduletest.Suite{
				Status: moduletest.Pass,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest": {
						Name:   "descriptive_test_name.tftest",
						Status: moduletest.Pass,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_two",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_three",
								Status: moduletest.Pass,
							},
						},
					},
					"other_descriptive_test_name.tftest": {
						Name:   "other_descriptive_test_name.tftest",
						Status: moduletest.Pass,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_two",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_three",
								Status: moduletest.Pass,
							},
						},
					},
				},
			},
			Expected: "\nSuccess! 6 passed, 0 failed.\n",
		},

		"passed and skipped tests": {
			Suite: &moduletest.Suite{
				Status: moduletest.Pass,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest": {
						Name:   "descriptive_test_name.tftest",
						Status: moduletest.Pass,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_two",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_three",
								Status: moduletest.Pass,
							},
						},
					},
					"other_descriptive_test_name.tftest": {
						Name:   "other_descriptive_test_name.tftest",
						Status: moduletest.Pass,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_two",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_three",
								Status: moduletest.Pass,
							},
						},
					},
				},
			},
			Expected: "\nSuccess! 4 passed, 0 failed, 2 skipped.\n",
		},

		"only failed tests": {
			Suite: &moduletest.Suite{
				Status: moduletest.Fail,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest": {
						Name:   "descriptive_test_name.tftest",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_three",
								Status: moduletest.Fail,
							},
						},
					},
					"other_descriptive_test_name.tftest": {
						Name:   "other_descriptive_test_name.tftest",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_three",
								Status: moduletest.Fail,
							},
						},
					},
				},
			},
			Expected: "\nFailure! 0 passed, 6 failed.\n",
		},

		"failed and skipped tests": {
			Suite: &moduletest.Suite{
				Status: moduletest.Fail,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest": {
						Name:   "descriptive_test_name.tftest",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_three",
								Status: moduletest.Fail,
							},
						},
					},
					"other_descriptive_test_name.tftest": {
						Name:   "other_descriptive_test_name.tftest",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_three",
								Status: moduletest.Skip,
							},
						},
					},
				},
			},
			Expected: "\nFailure! 0 passed, 4 failed, 2 skipped.\n",
		},

		"failed, passed and skipped tests": {
			Suite: &moduletest.Suite{
				Status: moduletest.Fail,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest": {
						Name:   "descriptive_test_name.tftest",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_three",
								Status: moduletest.Skip,
							},
						},
					},
					"other_descriptive_test_name.tftest": {
						Name:   "other_descriptive_test_name.tftest",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_two",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_three",
								Status: moduletest.Pass,
							},
						},
					},
				},
			},
			Expected: "\nFailure! 2 passed, 2 failed, 2 skipped.\n",
		},

		"failed and errored tests": {
			Suite: &moduletest.Suite{
				Status: moduletest.Error,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest": {
						Name:   "descriptive_test_name.tftest",
						Status: moduletest.Error,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Error,
							},
							{
								Name:   "test_three",
								Status: moduletest.Fail,
							},
						},
					},
					"other_descriptive_test_name.tftest": {
						Name:   "other_descriptive_test_name.tftest",
						Status: moduletest.Error,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Error,
							},
							{
								Name:   "test_three",
								Status: moduletest.Error,
							},
						},
					},
				},
			},
			Expected: "\nFailure! 0 passed, 6 failed.\n",
		},

		"failed, errored, passed, and skipped tests": {
			Suite: &moduletest.Suite{
				Status: moduletest.Error,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest": {
						Name:   "descriptive_test_name.tftest",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_two",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_three",
								Status: moduletest.Fail,
							},
						},
					},
					"other_descriptive_test_name.tftest": {
						Name:   "other_descriptive_test_name.tftest",
						Status: moduletest.Error,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Error,
							},
							{
								Name:   "test_two",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_three",
								Status: moduletest.Skip,
							},
						},
					},
				},
			},
			Expected: "\nFailure! 2 passed, 2 failed, 2 skipped.\n",
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {

			streams, done := terminal.StreamsForTesting(t)
			view := NewTest(arguments.ViewHuman, NewView(streams))

			view.Conclusion(tc.Suite)

			actual := done(t).Stdout()
			expected := tc.Expected
			if diff := cmp.Diff(expected, actual); len(diff) > 0 {
				t.Fatalf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
			}
		})
	}
}

func TestTestHuman_File(t *testing.T) {
	tcs := map[string]struct {
		File     *moduletest.File
		Expected string
	}{
		"pass": {
			File:     &moduletest.File{Name: "main.tf", Status: moduletest.Pass},
			Expected: "main.tf... pass\n",
		},

		"pending": {
			File:     &moduletest.File{Name: "main.tf", Status: moduletest.Pending},
			Expected: "main.tf... pending\n",
		},

		"skip": {
			File:     &moduletest.File{Name: "main.tf", Status: moduletest.Skip},
			Expected: "main.tf... skip\n",
		},

		"fail": {
			File:     &moduletest.File{Name: "main.tf", Status: moduletest.Fail},
			Expected: "main.tf... fail\n",
		},

		"error": {
			File:     &moduletest.File{Name: "main.tf", Status: moduletest.Error},
			Expected: "main.tf... fail\n",
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {

			streams, done := terminal.StreamsForTesting(t)
			view := NewTest(arguments.ViewHuman, NewView(streams))

			view.File(tc.File)

			actual := done(t).Stdout()
			expected := tc.Expected
			if diff := cmp.Diff(expected, actual); len(diff) > 0 {
				t.Fatalf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
			}
		})
	}
}

func TestTestHuman_Run(t *testing.T) {
	tcs := map[string]struct {
		Run    *moduletest.Run
		StdOut string
		StdErr string
	}{
		"pass": {
			Run:    &moduletest.Run{Name: "run_block", Status: moduletest.Pass},
			StdOut: "  run \"run_block\"... pass\n",
		},

		"pass_with_diags": {
			Run: &moduletest.Run{
				Name:        "run_block",
				Status:      moduletest.Pass,
				Diagnostics: tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Warning, "a warning occurred", "some warning happened during this test")},
			},
			StdOut: `  run "run_block"... pass

Warning: a warning occurred

some warning happened during this test
`,
		},

		"pending": {
			Run:    &moduletest.Run{Name: "run_block", Status: moduletest.Pending},
			StdOut: "  run \"run_block\"... pending\n",
		},

		"skip": {
			Run:    &moduletest.Run{Name: "run_block", Status: moduletest.Skip},
			StdOut: "  run \"run_block\"... skip\n",
		},

		"fail": {
			Run:    &moduletest.Run{Name: "run_block", Status: moduletest.Fail},
			StdOut: "  run \"run_block\"... fail\n",
		},

		"fail_with_diags": {
			Run: &moduletest.Run{
				Name:   "run_block",
				Status: moduletest.Fail,
				Diagnostics: tfdiags.Diagnostics{
					tfdiags.Sourceless(tfdiags.Error, "a comparison failed", "details details details"),
					tfdiags.Sourceless(tfdiags.Error, "a second comparison failed", "other details"),
				},
			},
			StdOut: "  run \"run_block\"... fail\n",
			StdErr: `
Error: a comparison failed

details details details

Error: a second comparison failed

other details
`,
		},

		"error": {
			Run:    &moduletest.Run{Name: "run_block", Status: moduletest.Error},
			StdOut: "  run \"run_block\"... fail\n",
		},

		"error_with_diags": {
			Run: &moduletest.Run{
				Name:        "run_block",
				Status:      moduletest.Error,
				Diagnostics: tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Error, "an error occurred", "something bad happened during this test")},
			},
			StdOut: "  run \"run_block\"... fail\n",
			StdErr: `
Error: an error occurred

something bad happened during this test
`,
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {

			streams, done := terminal.StreamsForTesting(t)
			view := NewTest(arguments.ViewHuman, NewView(streams))

			view.Run(tc.Run)

			output := done(t)
			actual, expected := output.Stdout(), tc.StdOut
			if diff := cmp.Diff(expected, actual); len(diff) > 0 {
				t.Errorf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
			}

			actual, expected = output.Stderr(), tc.StdErr
			if diff := cmp.Diff(expected, actual); len(diff) > 0 {
				t.Errorf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
			}
		})
	}
}
