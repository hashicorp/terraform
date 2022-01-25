package command

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/mitchellh/cli"
)

func TestVersionCommand_implements(t *testing.T) {
	var _ cli.Command = &VersionCommand{}
}

func TestVersion(t *testing.T) {
	td, err := ioutil.TempDir("", "terraform-test-version")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

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

	ui := cli.NewMockUi()
	c := &VersionCommand{
		Meta: Meta{
			Ui: ui,
		},
		Version:           "4.5.6",
		VersionPrerelease: "foo",
		Platform:          getproviders.Platform{OS: "aros", Arch: "riscv64"},
	}
	if err := c.replaceLockedDependencies(locks); err != nil {
		t.Fatal(err)
	}
	if code := c.Run([]string{}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	expected := "Terraform v4.5.6-foo\non aros_riscv64\n+ provider registry.terraform.io/hashicorp/test1 v7.8.9-beta.2\n+ provider registry.terraform.io/hashicorp/test2 v1.2.3"
	if actual != expected {
		t.Fatalf("wrong output\ngot:\n%s\nwant:\n%s", actual, expected)
	}

}

func TestVersion_flags(t *testing.T) {
	ui := new(cli.MockUi)
	m := Meta{
		Ui: ui,
	}

	// `terraform version`
	c := &VersionCommand{
		Meta:              m,
		Version:           "4.5.6",
		VersionPrerelease: "foo",
		Platform:          getproviders.Platform{OS: "aros", Arch: "riscv64"},
	}

	if code := c.Run([]string{"-v", "-version"}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	expected := "Terraform v4.5.6-foo\non aros_riscv64"
	if actual != expected {
		t.Fatalf("wrong output\ngot: %#v\nwant: %#v", actual, expected)
	}
}

func TestVersion_outdated(t *testing.T) {
	ui := new(cli.MockUi)
	m := Meta{
		Ui: ui,
	}

	c := &VersionCommand{
		Meta:      m,
		Version:   "4.5.6",
		CheckFunc: mockVersionCheckFunc(true, "4.5.7"),
		Platform:  getproviders.Platform{OS: "aros", Arch: "riscv64"},
	}

	if code := c.Run([]string{}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	expected := "Terraform v4.5.6\non aros_riscv64\n\nYour version of Terraform is out of date! The latest version\nis 4.5.7. You can update by downloading from https://www.terraform.io/downloads.html"
	if actual != expected {
		t.Fatalf("wrong output\ngot: %#v\nwant: %#v", actual, expected)
	}
}

func TestVersion_json(t *testing.T) {
	td, err := ioutil.TempDir("", "terraform-test-version")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := cli.NewMockUi()
	meta := Meta{
		Ui: ui,
	}

	// `terraform version -json` without prerelease
	c := &VersionCommand{
		Meta:     meta,
		Version:  "4.5.6",
		Platform: getproviders.Platform{OS: "aros", Arch: "riscv64"},
	}
	if code := c.Run([]string{"-json"}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
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

	// flush the output from the mock ui
	ui.OutputWriter.Reset()

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
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	actual = strings.TrimSpace(ui.OutputWriter.String())
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
	ui := new(cli.MockUi)
	m := Meta{
		Ui: ui,
	}

	c := &VersionCommand{
		Meta:      m,
		Version:   "4.5.6",
		CheckFunc: mockVersionCheckFunc(true, "4.5.7"),
		Platform:  getproviders.Platform{OS: "aros", Arch: "riscv64"},
	}

	if code := c.Run([]string{"-json"}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
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
