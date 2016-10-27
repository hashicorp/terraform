package getter

import (
	"testing"
	)

type testdata struct {
	input string
	expected string
}

var tests = []testdata {
	{ "git@github.com:org/project.git", "git::ssh://git@github.com/org/project.git" },
	{ "git@github.com:org/project.git?ref=test-branch", "git::ssh://git@github.com/org/project.git?ref=test-branch" },
	{ "git@github.com:org/project.git//module/a", "git::ssh://git@github.com/org/project.git//module/a" },
	{ "git@github.com:org/project.git//module/a?ref=test-branch", "git::ssh://git@github.com/org/project.git//module/a?ref=test-branch" },
	{ "git@github.xyz.com:org/project.git", "git::ssh://git@github.xyz.com/org/project.git" },
	{ "git@github.xyz.com:org/project.git?ref=test-branch", "git::ssh://git@github.xyz.com/org/project.git?ref=test-branch" },
	{ "git@github.xyz.com:org/project.git//module/a", "git::ssh://git@github.xyz.com/org/project.git//module/a" },
	{ "git@github.xyz.com:org/project.git//module/a?ref=test-branch", "git::ssh://git@github.xyz.com/org/project.git//module/a?ref=test-branch" },
	{ "github.com/hashicorp/terraform.git", "git::https://github.com/hashicorp/terraform.git" },
	{ "github.com/hashicorp/terraform", "git::https://github.com/hashicorp/terraform.git" },
	{ "github.com/hashicorp/terraform.git?ref=test-branch", "git::https://github.com/hashicorp/terraform.git?ref=test-branch" },
	{ "github.com/hashicorp/terraform.git//modules/a", "git::https://github.com/hashicorp/terraform.git///modules/a" },
	{ "github.com/hashicorp/terraform//modules/a", "git::https://github.com/hashicorp/terraform.git///modules/a" },
	{ "github.xyz.com/hashicorp/terraform.git", "git::https://github.xyz.com/hashicorp/terraform.git" },
	{ "github.xyz.com/hashicorp/terraform", "git::https://github.xyz.com/hashicorp/terraform.git" },
	{ "github.xyz.com/hashicorp/terraform.git?ref=test-branch", "git::https://github.xyz.com/hashicorp/terraform.git?ref=test-branch" },
	{ "github.xyz.com/hashicorp/terraform.git//modules/a", "git::https://github.xyz.com/hashicorp/terraform.git///modules/a" },
	{ "github.xyz.com/hashicorp/terraform//modules/a", "git::https://github.xyz.com/hashicorp/terraform.git///modules/a" },
}

func TestDetect(t *testing.T) {
	detector := new(GitHubDetector)
	for _, data := range tests {
		a, s, e := detector.Detect(data.input, "")
		if a != data.expected || e != nil || s != true {
			msg := a
			if a == "" {
				if e != nil {
					msg = e.Error()
				} else {
					msg = "false"
				}
			}
			t.Error(
				"For", data.input,
				"expected", data.expected,
				"got", msg,
			)
		}
	}
}
