package artifact

import (
	"testing"

	"github.com/hashicorp/terraform/internal/moduletest"
)

func Test_suiteFilesAsSortedList(t *testing.T) {
	cases := map[string]struct {
		Suite         *moduletest.Suite
		ExpectedNames map[int]string
	}{
		"no test files": {
			Suite: &moduletest.Suite{},
		},
		"3 test files ordered in map": {
			Suite: &moduletest.Suite{
				Status: moduletest.Skip,
				Files: map[string]*moduletest.File{
					"test_file_1.tftest.hcl": {
						Name:   "test_file_1.tftest.hcl",
						Status: moduletest.Skip,
						Runs:   []*moduletest.Run{},
					},
					"test_file_2.tftest.hcl": {
						Name:   "test_file_2.tftest.hcl",
						Status: moduletest.Skip,
						Runs:   []*moduletest.Run{},
					},
					"test_file_3.tftest.hcl": {
						Name:   "test_file_3.tftest.hcl",
						Status: moduletest.Skip,
						Runs:   []*moduletest.Run{},
					},
				},
			},
			ExpectedNames: map[int]string{
				0: "test_file_1.tftest.hcl",
				1: "test_file_2.tftest.hcl",
				2: "test_file_3.tftest.hcl",
			},
		},
		"3 test files unordered in map": {
			Suite: &moduletest.Suite{
				Status: moduletest.Skip,
				Files: map[string]*moduletest.File{
					"test_file_3.tftest.hcl": {
						Name:   "test_file_3.tftest.hcl",
						Status: moduletest.Skip,
						Runs:   []*moduletest.Run{},
					},
					"test_file_1.tftest.hcl": {
						Name:   "test_file_1.tftest.hcl",
						Status: moduletest.Skip,
						Runs:   []*moduletest.Run{},
					},
					"test_file_2.tftest.hcl": {
						Name:   "test_file_2.tftest.hcl",
						Status: moduletest.Skip,
						Runs:   []*moduletest.Run{},
					},
				},
			},
			ExpectedNames: map[int]string{
				0: "test_file_1.tftest.hcl",
				1: "test_file_2.tftest.hcl",
				2: "test_file_3.tftest.hcl",
			},
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			list := suiteFilesAsSortedList(tc.Suite.Files)

			if len(tc.ExpectedNames) != len(tc.Suite.Files) {
				t.Fatalf("expected there to be %d items, got %d", len(tc.ExpectedNames), len(tc.Suite.Files))
			}

			if len(tc.ExpectedNames) == 0 {
				return
			}

			for k, v := range tc.ExpectedNames {
				if list[k].Name != v {
					t.Fatalf("expected element %d in sorted list to be named %s, got %s", k, v, list[k].Name)
				}
			}
		})
	}
}
