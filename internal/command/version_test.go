// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/cli"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
)

func TestVersionCommand_implements(t *testing.T) {
	var _ cli.Command = &VersionCommand{}
}

func TestVersion(t *testing.T) {
	td := t.TempDir()
	t.Chdir(td)

	// We'll create a fixed dependency lock file in our working directory
	// so we can verify that the version command shows the information
	// from it.
	locks := depsfile.NewLocks()
	locks.SetProvider(
		addrs.NewDefaultProvider("test2"),
		getproviders.MustParseVersion("1.2.3"),
		nil,
		nil,
	)
	locks.SetProvider(
		addrs.NewDefaultProvider("test1"),
		getproviders.MustParseVersion("7.8.9-beta.2"),
		nil,
		nil,
	)

	view, done := testView(t)
	c := &VersionCommand{
		Meta: Meta{
			View: view,
		},
		Version:           "4.5.6",
		VersionPrerelease: "foo",
		Platform:          getproviders.Platform{OS: "aros", Arch: "riscv64"},
	}
	if err := c.replaceLockedDependencies(locks); err != nil {
		t.Fatal(err)
	}
	if code := c.Run([]string{}); code != 0 {
		t.Fatalf("bad: \n%s", done(t).All())
	}

	actual := strings.TrimSpace(done(t).All())
	expected := "Terraform v4.5.6-foo\non aros_riscv64\n+ provider registry.terraform.io/hashicorp/test1 v7.8.9-beta.2\n+ provider registry.terraform.io/hashicorp/test2 v1.2.3"
	if actual != expected {
		t.Fatalf("wrong output\ngot:\n%s\nwant:\n%s", actual, expected)
	}
}

// terraform version must support (but not use) -v and -version flags.
// This is because whenever a user runs `terraform <any command name> -version`, etc, main.go
// will call the version command with all of the supplied flags and arguments.
func TestVersion_flags(t *testing.T) {
	view, done := testView(t)
	m := Meta{
		View: view,
	}

	// `terraform version`
	c := &VersionCommand{
		Meta:              m,
		Version:           "4.5.6",
		VersionPrerelease: "foo",
		Platform:          getproviders.Platform{OS: "aros", Arch: "riscv64"},
	}

	if code := c.Run([]string{"-v", "-version", "--version"}); code != 0 {
		t.Fatalf("bad: \n%s", done(t).All())
	}

	actual := strings.TrimSpace(done(t).All())
	expected := "Terraform v4.5.6-foo\non aros_riscv64"
	if actual != expected {
		t.Fatalf("wrong output\ngot: %#v\nwant: %#v", actual, expected)
	}
}

func TestVersion_unexpectedArgsOrFlags(t *testing.T) {
	t.Run("unexpected positional arguments are ignored without error", func(t *testing.T) {
		view, done := testView(t)
		m := Meta{
			View: view,
		}

		// `terraform version`
		c := &VersionCommand{
			Meta:              m,
			Version:           "4.5.6",
			VersionPrerelease: "foo",
			Platform:          getproviders.Platform{OS: "aros", Arch: "riscv64"},
		}

		// Human output
		args := []string{
			"foo",
			"bar",
		}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: \n%s", done(t).All())
		}

		output := done(t)
		actual := strings.TrimSpace(output.Stdout())
		expected := "Terraform v4.5.6-foo\non aros_riscv64"
		if actual != expected {
			t.Fatalf("wrong stdout output\ngot: %#v\nwant: %#v", actual, expected)
		}

		actual = strings.TrimSpace(output.Stderr())
		expected = ""
		if actual != expected {
			t.Fatalf("wrong stderr output\ngot: %#v\nwant: %#v", actual, expected)
		}

		// Machine-readable / JSON output
		view, done = testView(t)
		c.Meta = Meta{
			View: view,
		}
		args = []string{
			"-json",
			"foo",
			"bar",
		}
		if code := c.Run(args); code != 0 {
			t.Fatalf("bad: \n%s", done(t).All())
		}

		output = done(t)
		actual = strings.TrimSpace(output.Stdout())
		expected = strings.TrimSpace(`
{
  "terraform_version": "4.5.6-foo",
  "platform": "aros_riscv64",
  "provider_selections": {},
  "terraform_outdated": false
}
`)
		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Fatalf("wrong output\n%s", diff)
		}

		actual = strings.TrimSpace(output.Stderr())
		expected = ""
		if actual != expected {
			t.Fatalf("wrong stderr output\ngot: %#v\nwant: %#v", actual, expected)
		}
	})

	t.Run("incorrect flag", func(t *testing.T) {
		view, done := testView(t)
		m := Meta{
			View: view,
		}

		// `terraform version`
		c := &VersionCommand{
			Meta:              m,
			Version:           "4.5.6",
			VersionPrerelease: "foo",
			Platform:          getproviders.Platform{OS: "aros", Arch: "riscv64"},
		}

		// Human output
		args := []string{
			"-no-color",
			"-foobar",
		}
		if code := c.Run(args); code != 1 {
			output := done(t)
			t.Fatalf("expected code 1 and error output, but got code %d:\nstdout: %s\nstderr: %s", code, output.Stdout(), output.Stderr())
		}

		output := done(t)
		actual := strings.TrimSpace(output.Stdout())
		expected := ""
		if actual != expected {
			t.Fatalf("wrong stdout output\ngot: %#v\nwant: %#v", actual, expected)
		}

		actual = strings.TrimSpace(output.Stderr())
		expected = `Error: Failed to parse command-line flags

flag provided but not defined: -foobar`
		if actual != expected {
			t.Fatalf("wrong stderr output\ngot: %#v\nwant: %#v", actual, expected)
		}

		// Machine-readable / JSON output
		view, done = testView(t)
		c.Meta = Meta{
			View: view,
		}
		args = []string{
			"-no-color",
			"-json",
			"-foobar",
		}
		if code := c.Run(args); code != 1 {
			t.Fatalf("expected code 1 and error output, but got code %d:\nstdout: %s\nstderr: %s", code, done(t).Stdout(), done(t).Stderr())
		}

		output = done(t)
		actual = strings.TrimSpace(output.Stdout())
		expected = ""
		if actual != expected {
			t.Fatalf("wrong stdout output\ngot: %#v\nwant: %#v", actual, expected)
		}

		// Human error output is rendered despite -json flag when an error occurs
		actual = strings.TrimSpace(output.Stderr())
		expected = `Error: Failed to parse command-line flags

flag provided but not defined: -foobar`
		if actual != expected {
			t.Fatalf("wrong stderr output\ngot: %#v\nwant: %#v", actual, expected)
		}
	})
}

func TestVersion_outdated(t *testing.T) {
	view, done := testView(t)
	m := Meta{
		View: view,
	}

	c := &VersionCommand{
		Meta:      m,
		Version:   "4.5.6",
		CheckFunc: mockVersionCheckFunc(true, "4.5.7"),
		Platform:  getproviders.Platform{OS: "aros", Arch: "riscv64"},
	}

	if code := c.Run([]string{}); code != 0 {
		t.Fatalf("bad: \n%s", done(t).All())
	}

	actual := strings.TrimSpace(done(t).All())
	expected := "Terraform v4.5.6\non aros_riscv64\n\nYour version of Terraform is out of date! The latest version\nis 4.5.7. You can update by downloading from https://developer.hashicorp.com/terraform/install"
	if actual != expected {
		t.Fatalf("wrong output\ngot: %#v\nwant: %#v", actual, expected)
	}
}

func TestVersion_json(t *testing.T) {
	td := t.TempDir()
	t.Chdir(td)

	view, done := testView(t)
	meta := Meta{
		View: view,
	}

	// `terraform version -json` without prerelease
	c := &VersionCommand{
		Meta:     meta,
		Version:  "4.5.6",
		Platform: getproviders.Platform{OS: "aros", Arch: "riscv64"},
	}
	if code := c.Run([]string{"-json"}); code != 0 {
		t.Fatalf("bad: \n%s", done(t).All())
	}

	actual := strings.TrimSpace(done(t).All())
	expected := strings.TrimSpace(`
{
  "terraform_version": "4.5.6",
  "platform": "aros_riscv64",
  "provider_selections": {},
  "terraform_outdated": false
}
`)
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Fatalf("wrong output\n%s", diff)
	}

	// Now we'll create a fixed dependency lock file in our working directory
	// so we can verify that the version command shows the information
	// from it.
	locks := depsfile.NewLocks()
	locks.SetProvider(
		addrs.NewDefaultProvider("test2"),
		getproviders.MustParseVersion("1.2.3"),
		nil,
		nil,
	)
	locks.SetProvider(
		addrs.NewDefaultProvider("test1"),
		getproviders.MustParseVersion("7.8.9-beta.2"),
		nil,
		nil,
	)

	// `terraform version -json` with prerelease and provider dependencies
	view, done = testView(t)
	meta.View = view
	c = &VersionCommand{
		Meta:              meta,
		Version:           "4.5.6",
		VersionPrerelease: "foo",
		Platform:          getproviders.Platform{OS: "aros", Arch: "riscv64"},
	}
	if err := c.replaceLockedDependencies(locks); err != nil {
		t.Fatal(err)
	}
	if code := c.Run([]string{"-json"}); code != 0 {
		t.Fatalf("bad: \n%s", done(t).All())
	}

	actual = strings.TrimSpace(done(t).All())
	expected = strings.TrimSpace(`
{
  "terraform_version": "4.5.6-foo",
  "platform": "aros_riscv64",
  "provider_selections": {
    "registry.terraform.io/hashicorp/test1": "7.8.9-beta.2",
    "registry.terraform.io/hashicorp/test2": "1.2.3"
  },
  "terraform_outdated": false
}
`)
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Fatalf("wrong output\n%s", diff)
	}
}

func TestVersion_jsonoutdated(t *testing.T) {
	view, done := testView(t)
	m := Meta{
		View: view,
	}

	c := &VersionCommand{
		Meta:      m,
		Version:   "4.5.6",
		CheckFunc: mockVersionCheckFunc(true, "4.5.7"),
		Platform:  getproviders.Platform{OS: "aros", Arch: "riscv64"},
	}

	if code := c.Run([]string{"-json"}); code != 0 {
		t.Fatalf("bad: \n%s", done(t).All())
	}

	actual := strings.TrimSpace(done(t).All())
	expected := "{\n  \"terraform_version\": \"4.5.6\",\n  \"platform\": \"aros_riscv64\",\n  \"provider_selections\": {},\n  \"terraform_outdated\": true\n}"
	if actual != expected {
		t.Fatalf("wrong output\ngot: %#v\nwant: %#v", actual, expected)
	}
}

func mockVersionCheckFunc(outdated bool, latest string) VersionCheckFunc {
	return func() (VersionCheckInfo, error) {
		return VersionCheckInfo{
			Outdated: outdated,
			Latest:   latest,
			// Alerts is not used by version command
		}, nil
	}
}
