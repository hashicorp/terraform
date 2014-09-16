package module

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var testHasGit bool

func init() {
	if _, err := exec.LookPath("git"); err == nil {
		testHasGit = true
	}
}

func TestGitGetter_impl(t *testing.T) {
	var _ Getter = new(GitGetter)
}

func TestGitGetter(t *testing.T) {
	if !testHasGit {
		t.Log("git not found, skipping")
		t.Skip()
	}

	g := new(GitGetter)
	dst := tempDir(t)

	// Git doesn't allow nested ".git" directories so we do some hackiness
	// here to get around that...
	moduleDir := filepath.Join(fixtureDir, "basic-git")
	oldName := filepath.Join(moduleDir, "DOTgit")
	newName := filepath.Join(moduleDir, ".git")
	if err := os.Rename(oldName, newName); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Rename(newName, oldName)

	// With a dir that doesn't exist
	if err := g.Get(dst, testModuleURL("basic-git")); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the main file exists
	mainPath := filepath.Join(dst, "main.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestGitGetter_branch(t *testing.T) {
	if !testHasGit {
		t.Log("git not found, skipping")
		t.Skip()
	}

	g := new(GitGetter)
	dst := tempDir(t)

	// Git doesn't allow nested ".git" directories so we do some hackiness
	// here to get around that...
	moduleDir := filepath.Join(fixtureDir, "basic-git")
	oldName := filepath.Join(moduleDir, "DOTgit")
	newName := filepath.Join(moduleDir, ".git")
	if err := os.Rename(oldName, newName); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Rename(newName, oldName)

	url := testModuleURL("basic-git")
	q := url.Query()
	q.Add("ref", "test-branch")
	url.RawQuery = q.Encode()

	if err := g.Get(dst, url); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the main file exists
	mainPath := filepath.Join(dst, "main_branch.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Get again should work
	if err := g.Get(dst, url); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the main file exists
	mainPath = filepath.Join(dst, "main_branch.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestGitGetter_tag(t *testing.T) {
	if !testHasGit {
		t.Log("git not found, skipping")
		t.Skip()
	}

	g := new(GitGetter)
	dst := tempDir(t)

	// Git doesn't allow nested ".git" directories so we do some hackiness
	// here to get around that...
	moduleDir := filepath.Join(fixtureDir, "basic-git")
	oldName := filepath.Join(moduleDir, "DOTgit")
	newName := filepath.Join(moduleDir, ".git")
	if err := os.Rename(oldName, newName); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Rename(newName, oldName)

	url := testModuleURL("basic-git")
	q := url.Query()
	q.Add("ref", "v1.0")
	url.RawQuery = q.Encode()

	if err := g.Get(dst, url); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the main file exists
	mainPath := filepath.Join(dst, "main_tag1.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Get again should work
	if err := g.Get(dst, url); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the main file exists
	mainPath = filepath.Join(dst, "main_tag1.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}
}
