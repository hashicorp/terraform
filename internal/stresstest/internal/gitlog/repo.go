package gitlog

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/apparentlymart/go-mingit/mingit"
)

// InitRepository initializes the given directory as a git repository, laid out
// so that it can have a work tree added after the caller has had an opportunity
// to prepare something for HEAD to refer to.
//
// The underlying "mingit" library creates bare repositories, so this wrapper
// is responsible for creating a suitable layout for a repository with a work
// tree, although it doesn't actually create the work tree yet. Once you've
// used the returned repository object to make the HEAD ref point somewhere,
// use CreateRepositoryWorkTree to convert the bare repository into a
// non-bare one.
//
// If the target directory already has a .git subdirectory inside it, this
// function will attempt to remove that directory and put a new one in its
// place. However, it'll leave the containing directory untouched, to avoid
// causing problems for any shells that might have it selected as current
// working directory.
func InitRepository(baseDir string) (*mingit.Repository, error) {
	gitDir := filepath.Join(baseDir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		err = os.RemoveAll(gitDir)
		if err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		return nil, err
	}

	return mingit.NewRepository(gitDir)
}

// CreateRepositoryWorkTree takes a directory previously passed to a successful
// InitRepository and tries to make its work tree match its current HEAD
// commit.
//
// All of the other git-related functionality in this package is done directly
// in Go code, but this particular operation works by running the normal "git"
// CLI as a subprocess, and so it'll fail if git isn't available in the PATH.
func CreateRepositoryWorkTree(baseDir string) error {
	// First we need to overwrite the configuration that mingit generated,
	// which marks the repository as being a bare repository.
	err := ioutil.WriteFile(filepath.Join(baseDir, ".git", "config"), repoConfig, 0644)
	if err != nil {
		return err
	}
	// Now we'll do our one operation that we must delegate to the normal git
	// executable, because mingit doesn't understand work trees and indices.
	// The "git read-tree" command is a plumbing command which reads the
	// content of a particular tree into the index. The -u option then
	// additionally copies the contents of the index into the work tree, which
	// should ultimately leave us in the typical state for a normal git
	// repository that doesn't have any work in progress.
	cmd := exec.Command(
		"git", "read-tree",
		"--reset",   // discard any uncommitted changes already in the index
		"-u",        // update the work tree to match the updated index
		"--trivial", // don't try to do any fancy merge stuff
		"--quiet",   // we're not going to listen to git's output anyway
		"HEAD",      // use the tree associated with the HEAD commit
	)
	cmd.Env = []string{
		"GITDIR=" + filepath.Join(baseDir, ".git"),
		"GIT_WORK_TREE=" + baseDir,
	}
	cmd.Dir = baseDir
	return cmd.Run()
}

var repoConfig = []byte(`[core]
	repositoryformatversion = 0
	bare = false
`)
