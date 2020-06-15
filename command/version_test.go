package command

import (
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/copy"
	"github.com/mitchellh/cli"
)

func TestVersionCommand_implements(t *testing.T) {
	var _ cli.Command = &VersionCommand{}
}

func TestVersion(t *testing.T) {
	fixtureDir := "testdata/providers-schema/basic"
	td := tempDir(t)
	copy.CopyDir(fixtureDir, td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": []string{"1.2.3"},
	})
	defer close()

	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
		ProviderSource:   providerSource,
	}

	// `terrafrom init`
	ic := &InitCommand{
		Meta: m,
	}
	if code := ic.Run([]string{}); code != 0 {
		t.Fatalf("init failed\n%s", ui.ErrorWriter)
	}
	// flush the init output from the mock ui
	ui.OutputWriter.Reset()

	// `terraform version`
	c := &VersionCommand{
		Meta:              m,
		Version:           "4.5.6",
		VersionPrerelease: "foo",
	}
	if code := c.Run([]string{}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	expected := "Terraform v4.5.6-foo\n+ provider registry.terraform.io/hashicorp/test v1.2.3"
	if actual != expected {
		t.Fatalf("wrong output\ngot: %#v\nwant: %#v", actual, expected)
	}

}

func TestVersion_json(t *testing.T) {
	fixtureDir := "testdata/providers-schema/basic"
	td := tempDir(t)
	copy.CopyDir(fixtureDir, td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": []string{"1.2.3"},
	})
	defer close()

	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
		ProviderSource:   providerSource,
	}

	// `terrafrom init`
	ic := &InitCommand{
		Meta: m,
	}
	if code := ic.Run([]string{}); code != 0 {
		t.Fatalf("init failed\n%s", ui.ErrorWriter)
	}
	// flush the init output from the mock ui
	ui.OutputWriter.Reset()

	// `terraform version -json` without prerelease
	c := &VersionCommand{
		Meta:    m,
		Version: "4.5.6",
	}
	if code := c.Run([]string{"-json"}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	expected := "{\n  \"terraform_version\": \"4.5.6\",\n  \"terraform_revision\": \"\",\n  \"provider_selections\": {\n    \"registry.terraform.io/hashicorp/test\": \"1.2.3\"\n  },\n  \"terraform_outdated\": false\n}"
	if actual != expected {
		t.Fatalf("wrong output\ngot: %#v\nwant: %#v", actual, expected)
	}

	// flush the output from the mock ui
	ui.OutputWriter.Reset()

	// `terraform version -json` with prerelease
	c = &VersionCommand{
		Meta:              m,
		Version:           "4.5.6",
		VersionPrerelease: "foo",
	}
	if code := c.Run([]string{"-json"}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	actual = strings.TrimSpace(ui.OutputWriter.String())
	expected = "{\n  \"terraform_version\": \"4.5.6-foo\",\n  \"terraform_revision\": \"\",\n  \"provider_selections\": {\n    \"registry.terraform.io/hashicorp/test\": \"1.2.3\"\n  },\n  \"terraform_outdated\": false\n}"
	if actual != expected {
		t.Fatalf("wrong output\ngot: %#v\nwant: %#v", actual, expected)
	}

}
