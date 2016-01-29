package archive

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func setupGitFixtures(t *testing.T) (string, func()) {
	testDir := testFixture("archive-git")
	oldName := filepath.Join(testDir, "DOTgit")
	newName := filepath.Join(testDir, ".git")

	cleanup := func() {
		os.Rename(newName, oldName)
		// Windows leaves an empty folder lying around afterward
		if runtime.GOOS == "windows" {
			os.Remove(newName)
		}
	}

	// We call this BEFORE and after each setup for tests that make lower-level
	// calls like runCommand
	cleanup()

	if err := os.Rename(oldName, newName); err != nil {
		t.Fatal(err)
	}

	return testDir, cleanup
}

func TestVCSPreflight(t *testing.T) {
	if !testHasGit {
		t.Skip("git not found")
	}

	testDir, cleanup := setupGitFixtures(t)
	defer cleanup()

	if err := vcsPreflight(testDir); err != nil {
		t.Fatal(err)
	}
}

func TestGitBranch(t *testing.T) {
	if !testHasGit {
		t.Skip("git not found")
	}

	testDir, cleanup := setupGitFixtures(t)
	defer cleanup()

	branch, err := gitBranch(testDir)
	if err != nil {
		t.Fatal(err)
	}

	expected := "master"
	if branch != expected {
		t.Fatalf("expected %q to be %q", branch, expected)
	}
}

func TestGitBranch_detached(t *testing.T) {
	if !testHasGit {
		t.Skip("git not found")
	}

	testDir := testFixture("archive-git")
	oldName := filepath.Join(testDir, "DOTgit")
	newName := filepath.Join(testDir, ".git")
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %#v", err)
	}

	// Copy and then remove the .git dir instead of moving and replacing like
	// other tests, since the checkout below is going to write to the reflog and
	// the index
	runCommand(t, pwd, "cp", "-r", oldName, newName)
	defer runCommand(t, pwd, "rm", "-rf", newName)

	runCommand(t, testDir, "git", "checkout", "--detach")

	branch, err := gitBranch(testDir)
	if err != nil {
		t.Fatal(err)
	}

	if branch != "" {
		t.Fatalf("expected branch to be empty, but it was: %s", branch)
	}
}

func TestGitCommit(t *testing.T) {
	if !testHasGit {
		t.Skip("git not found")
	}

	testDir, cleanup := setupGitFixtures(t)
	defer cleanup()

	commit, err := gitCommit(testDir)
	if err != nil {
		t.Fatal(err)
	}

	expected := "7525d17cbbb56f3253a20903ffddc07c6c935c76"
	if commit != expected {
		t.Fatalf("expected %q to be %q", commit, expected)
	}
}

func TestGitRemotes(t *testing.T) {
	if !testHasGit {
		t.Skip("git not found")
	}

	testDir, cleanup := setupGitFixtures(t)
	defer cleanup()

	remotes, err := gitRemotes(testDir)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]string{
		"remote.origin":   "https://github.com/hashicorp/origin.git",
		"remote.upstream": "https://github.com/hashicorp/upstream.git",
	}

	if !reflect.DeepEqual(remotes, expected) {
		t.Fatalf("expected %+v to be %+v", remotes, expected)
	}
}

func TestVCSMetadata_git(t *testing.T) {
	if !testHasGit {
		t.Skip("git not found")
	}

	testDir, cleanup := setupGitFixtures(t)
	defer cleanup()

	metadata, err := vcsMetadata(testDir)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]string{
		"branch":          "master",
		"commit":          "7525d17cbbb56f3253a20903ffddc07c6c935c76",
		"remote.origin":   "https://github.com/hashicorp/origin.git",
		"remote.upstream": "https://github.com/hashicorp/upstream.git",
	}

	if !reflect.DeepEqual(metadata, expected) {
		t.Fatalf("expected %+v to be %+v", metadata, expected)
	}
}

func TestVCSMetadata_git_detached(t *testing.T) {
	if !testHasGit {
		t.Skip("git not found")
	}

	testDir := testFixture("archive-git")
	oldName := filepath.Join(testDir, "DOTgit")
	newName := filepath.Join(testDir, ".git")
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %#v", err)
	}

	// Copy and then remove the .git dir instead of moving and replacing like
	// other tests, since the checkout below is going to write to the reflog and
	// the index
	runCommand(t, pwd, "cp", "-r", oldName, newName)
	defer runCommand(t, pwd, "rm", "-rf", newName)

	runCommand(t, testDir, "git", "checkout", "--detach")

	metadata, err := vcsMetadata(testDir)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]string{
		"branch":          "",
		"commit":          "7525d17cbbb56f3253a20903ffddc07c6c935c76",
		"remote.origin":   "https://github.com/hashicorp/origin.git",
		"remote.upstream": "https://github.com/hashicorp/upstream.git",
	}

	if !reflect.DeepEqual(metadata, expected) {
		t.Fatalf("expected %+v to be %+v", metadata, expected)
	}
}

func TestVCSPathDetect_git(t *testing.T) {
	testDir, cleanup := setupGitFixtures(t)
	defer cleanup()

	vcs, err := vcsDetect(testDir)
	if err != nil {
		t.Errorf("VCS detection failed")
	}

	if vcs.Name != "git" {
		t.Errorf("Expected to find git; found %s", vcs.Name)
	}
}

func TestVCSPathDetect_git_failure(t *testing.T) {
	_, err := vcsDetect(testFixture("archive-flat"))
	// We expect to get an error because there is no git repo here
	if err == nil {
		t.Errorf("VCS detection failed")
	}
}

func TestVCSPathDetect_hg(t *testing.T) {
	vcs, err := vcsDetect(testFixture("archive-hg"))
	if err != nil {
		t.Errorf("VCS detection failed")
	}

	if vcs.Name != "hg" {
		t.Errorf("Expected to find hg; found %s", vcs.Name)
	}
}

func TestVCSPathDetect_hg_absolute(t *testing.T) {
	abspath, err := filepath.Abs(testFixture("archive-hg"))
	vcs, err := vcsDetect(abspath)
	if err != nil {
		t.Errorf("VCS detection failed")
	}

	if vcs.Name != "hg" {
		t.Errorf("Expected to find hg; found %s", vcs.Name)
	}
}

func runCommand(t *testing.T, path, command string, args ...string) {
	var stderr, stdout bytes.Buffer
	cmd := exec.Command(command, args...)
	cmd.Dir = path
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("error running command: %s\nstdout: %s\nstderr: %s",
			err, stdout.String(), stderr.String())
	}
}
