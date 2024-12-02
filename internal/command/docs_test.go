package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/cli"
)

func TestDocsCommand_implements(t *testing.T) {
	var _ cli.Command = &CommandDocs{}
}

func TestDocs_basic(t *testing.T) {
	td := t.TempDir()
	defer testChdir(t, td)()

	// Create a mock provider documentation structure
	docsDir := filepath.Join(td, ".terraform", "docs", "test")
	mockDocs := map[string]string{
		"docs/resources/example.md": `# Resource: test_example
This is an example resource.
## Arguments
* arg1 - (Required) First argument
## Attributes
* id - The ID of the resource`,
		"docs/data-sources/scaffolding_data_source.md": `# Data Source: test_example
This is an example data source.
## Arguments
* name - (Required) The name to lookup`,
	}

	// Create the mock documentation files
	for path, content := range mockDocs {
		fullPath := filepath.Join(docsDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Create mock lock file
	lockContent := `provider "registry.terraform.io/hashicorp/scaffolding" {
		version = "1.0.0"
	}`
	err := os.WriteFile(".terraform.lock.hcl", []byte(lockContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	ui := cli.NewMockUi()
	c := &CommandDocs{
		Meta: Meta{
			Ui: ui,
		},
	}

	// Test listing resources
	if code := c.Run([]string{"scaffolding", "-l"}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	expected := strings.TrimSpace(`
Resources:
* resource

Data Sources:
* data_source`)
	if actual != expected {
		t.Fatalf("wrong output\ngot:\n%s\nwant:\n%s", actual, expected)
	}
}

func TestDocs_resourceDoc(t *testing.T) {
	td := t.TempDir()
	defer testChdir(t, td)()

	// Create test documentation
	docsDir := filepath.Join(td, ".terraform", "docs", "scaffolding")
	resourceContent := `# Resource: scaffolding_example
This is an scaffolding resource.
## Arguments
* arg1 - (Required) First argument`

	err := os.MkdirAll(filepath.Join(docsDir, "docs", "resources"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(
		filepath.Join(docsDir, "docs", "resources", "example.md"),
		[]byte(resourceContent),
		0644,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Create mock lock file
	lockContent := `provider "registry.terraform.io/hashicorp/scaffolding" {
		version = "1.0.0"
	}`
	err = os.WriteFile(".terraform.lock.hcl", []byte(lockContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	ui := cli.NewMockUi()
	c := &CommandDocs{
		Meta: Meta{
			Ui: ui,
		},
	}

	// Test showing resource documentation
	if code := c.Run([]string{"scaffolding", "example", "-r"}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	expected := strings.TrimSpace(resourceContent)
	if actual != expected {
		t.Fatalf("wrong output\ngot:\n%s\nwant:\n%s", actual, expected)
	}
}

func TestDocs_searchKeyword(t *testing.T) {
	td := t.TempDir()
	defer testChdir(t, td)()

	// Create test documentation
	docsDir := filepath.Join(td, ".terraform", "docs", "scaffolding")
	resourceContent := `# Resource: test_example
This is an example resource.
## Arguments
* arg1 - (Required) First argument
## Attributes
* id - The ID of the resource`

	err := os.MkdirAll(filepath.Join(docsDir, "docs", "resources"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(
		filepath.Join(docsDir, "docs", "resources", "example.md"),
		[]byte(resourceContent),
		0644,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Create mock lock file
	lockContent := `provider "registry.terraform.io/hashicorp/scaffolding" {
		version = "1.0.0"
	}`
	err = os.WriteFile(".terraform.lock.hcl", []byte(lockContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	ui := cli.NewMockUi()
	c := &CommandDocs{
		Meta: Meta{
			Ui: ui,
		},
	}

	// Test searching for a specific section
	if code := c.Run([]string{"test", "example", "-r", "Arguments"}); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(ui.OutputWriter.String())
	expected := strings.TrimSpace(`=== Section matching 'Arguments' ===
## Arguments
* arg1 - (Required) First argument
=== End of section ===`)
	if actual != expected {
		t.Fatalf("wrong output\ngot:\n%s\nwant:\n%s", actual, expected)
	}
}

func TestDocs_invalidProvider(t *testing.T) {
	ui := cli.NewMockUi()
	c := &CommandDocs{
		Meta: Meta{
			Ui: ui,
		},
	}

	if code := c.Run([]string{"nonexistent"}); code == 0 {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestDocs_missingDocs(t *testing.T) {
	td := t.TempDir()
	defer testChdir(t, td)()

	// Create mock lock file without docs
	lockContent := `provider "registry.terraform.io/hashicorp/scaffolding" {
		version = "1.0.0"
	}`
	err := os.WriteFile(".terraform.lock.hcl", []byte(lockContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	ui := cli.NewMockUi()
	c := &CommandDocs{
		Meta: Meta{
			Ui: ui,
		},
	}

	if code := c.Run([]string{"test", "nonexistent", "-r"}); code == 0 {
		t.Fatal("expected error for nonexistent resource")
	}
}

func TestDocs_invalidFlags(t *testing.T) {
	ui := cli.NewMockUi()
	c := &CommandDocs{
		Meta: Meta{
			Ui: ui,
		},
	}

	if code := c.Run([]string{"test", "example", "-invalid"}); code == 0 {
		t.Fatal("expected error for invalid flag")
	}
}
