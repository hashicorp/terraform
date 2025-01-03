package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/cli"
)

// testCommandDocs creates a mock docs command with test directories set up
type testCommandDocs struct {
	*CommandDocs
	tempDir string
	ui      *cli.MockUi
	cleanup func()
}

// setupTestDocs creates and sets up mock documentation structure
func setupTestDocs(t *testing.T) *testCommandDocs {
	td := t.TempDir()
	ui := cli.NewMockUi()

	// Create lock file
	lockContent := `provider "registry.terraform.io/hashicorp/test" {
  version = "1.0.0"
}`
	err := os.WriteFile(filepath.Join(td, ".terraform.lock.hcl"), []byte(lockContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create docs directory structure
	docsDir := filepath.Join(td, ".terraform", "docs", "test")
	mockDocs := map[string]string{
		"docs/resources/example.md": `# Resource: test_example
This is an example resource.
## Arguments
* arg1 - (Required) First argument
## Attributes
* id - The ID of the resource`,
		"docs/data-sources/example.md": `# Data Source: test_example
This is an example data source.
## Arguments
* name - (Required) The name to lookup`,
	}

	for path, content := range mockDocs {
		fullPath := filepath.Join(docsDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Add version file to prevent re-cloning
	if err := os.WriteFile(filepath.Join(docsDir, ".version"), []byte("1.0.0"), 0644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// Change to test directory
	if err := os.Chdir(td); err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		if err := os.Chdir(origDir); err != nil {
			t.Error(err)
		}
	}

	meta := Meta{
		Ui: ui,
	}

	return &testCommandDocs{
		CommandDocs: &CommandDocs{
			Meta: meta,
		},
		tempDir: td,
		ui:      ui,
		cleanup: cleanup,
	}
}

func TestDocsCommand_implements(t *testing.T) {
	var _ cli.Command = &CommandDocs{}
}

func TestDocs_basic(t *testing.T) {
	cmd := setupTestDocs(t)
	defer cmd.cleanup()

	// Test listing resources
	if code := cmd.Run([]string{"test", "-l"}); code != 0 {
		t.Fatalf("bad: \n%s", cmd.ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(cmd.ui.OutputWriter.String())
	expected := strings.TrimSpace(`
Resources:
* example

Data Sources:
* example`)

	if actual != expected {
		t.Fatalf("wrong output\ngot:\n%s\nwant:\n%s", actual, expected)
	}
}

func TestDocs_resourceDoc(t *testing.T) {
	cmd := setupTestDocs(t)
	defer cmd.cleanup()

	// Test showing resource documentation
	if code := cmd.Run([]string{"test", "example", "-r"}); code != 0 {
		t.Fatalf("bad: \n%s", cmd.ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(cmd.ui.OutputWriter.String())
	expected := strings.TrimSpace(`# Resource: test_example
This is an example resource.
## Arguments
* arg1 - (Required) First argument
## Attributes
* id - The ID of the resource`)

	if actual != expected {
		t.Fatalf("wrong output\ngot:\n%s\nwant:\n%s", actual, expected)
	}
}

func TestDocs_searchKeyword(t *testing.T) {
	cmd := setupTestDocs(t)
	defer cmd.cleanup()

	// Test searching for a specific section
	if code := cmd.Run([]string{"test", "example", "-r", "Arguments"}); code != 0 {
		t.Fatalf("bad: \n%s", cmd.ui.ErrorWriter.String())
	}

	actual := strings.TrimSpace(cmd.ui.OutputWriter.String())
	expected := strings.TrimSpace(`
## Arguments
* arg1 - (Required) First argument
`)

	if actual != expected {
		t.Fatalf("wrong output\ngot:\n%s\nwant:\n%s", actual, expected)
	}
}

func TestDocs_invalidProvider(t *testing.T) {
	cmd := setupTestDocs(t)
	defer cmd.cleanup()

	if code := cmd.Run([]string{"nonexistent"}); code == 0 {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestDocs_missingDocs(t *testing.T) {
	cmd := setupTestDocs(t)
	defer cmd.cleanup()

	if code := cmd.Run([]string{"test", "nonexistent", "-r"}); code == 0 {
		t.Fatal("expected error for nonexistent resource")
	}
}

func TestDocs_invalidFlags(t *testing.T) {
	cmd := setupTestDocs(t)
	defer cmd.cleanup()

	if code := cmd.Run([]string{"test", "example", "-invalid"}); code == 0 {
		t.Fatal("expected error for invalid flag")
	}
}
