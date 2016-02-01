package archive

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	version "github.com/hashicorp/go-version"
)

// VCS is a struct that explains how to get the file list for a given
// VCS.
type VCS struct {
	Name string

	// Detect is a list of files/folders that if they exist, signal that
	// this VCS is the VCS in use.
	Detect []string

	// Files returns the files that are under version control for the
	// given path.
	Files VCSFilesFunc

	// Metadata returns arbitrary metadata about the underlying VCS for the
	// given path.
	Metadata VCSMetadataFunc

	// Preflight is a function to run before looking for VCS files.
	Preflight VCSPreflightFunc
}

// VCSList is the list of VCS we recognize.
var VCSList = []*VCS{
	&VCS{
		Name:      "git",
		Detect:    []string{".git/"},
		Preflight: gitPreflight,
		Files:     vcsFilesCmd("git", "ls-files"),
		Metadata:  gitMetadata,
	},
	&VCS{
		Name:   "hg",
		Detect: []string{".hg/"},
		Files:  vcsTrimCmd(vcsFilesCmd("hg", "locate", "-f", "--include", ".")),
	},
	&VCS{
		Name:   "svn",
		Detect: []string{".svn/"},
		Files:  vcsFilesCmd("svn", "ls"),
	},
}

// VCSFilesFunc is the callback invoked to return the files in the VCS.
//
// The return value should be paths relative to the given path.
type VCSFilesFunc func(string) ([]string, error)

// VCSMetadataFunc is the callback invoked to get arbitrary information about
// the current VCS.
//
// The return value should be a map of key-value pairs.
type VCSMetadataFunc func(string) (map[string]string, error)

// VCSPreflightFunc is a function that runs before VCS detection to be
// configured by the user. It may be used to check if pre-requisites (like the
// actual VCS) are installed or that a program is at the correct version. If an
// error is returned, the VCS will not be processed and the error will be
// returned up the stack.
//
// The given argument is the path where the VCS is running.
type VCSPreflightFunc func(string) error

// vcsDetect detects the VCS that is used for path.
func vcsDetect(path string) (*VCS, error) {
	dir := path
	for {
		for _, v := range VCSList {
			for _, f := range v.Detect {
				check := filepath.Join(dir, f)
				if _, err := os.Stat(check); err == nil {
					return v, nil
				}
			}
		}
		lastDir := dir
		dir = filepath.Dir(dir)
		if dir == lastDir {
			break
		}
	}

	return nil, fmt.Errorf("no VCS found for path: %s", path)
}

// vcsPreflight returns the metadata for the VCS directory path.
func vcsPreflight(path string) error {
	vcs, err := vcsDetect(path)
	if err != nil {
		return fmt.Errorf("error detecting VCS: %s", err)
	}

	if vcs.Preflight != nil {
		return vcs.Preflight(path)
	}

	return nil
}

// vcsFiles returns the files for the VCS directory path.
func vcsFiles(path string) ([]string, error) {
	vcs, err := vcsDetect(path)
	if err != nil {
		return nil, fmt.Errorf("error detecting VCS: %s", err)
	}

	if vcs.Files != nil {
		return vcs.Files(path)
	}

	return nil, nil
}

// vcsFilesCmd creates a Files-compatible function that reads the files
// by executing the command in the repository path and returning each
// line in stdout.
func vcsFilesCmd(args ...string) VCSFilesFunc {
	return func(path string) ([]string, error) {
		var stderr, stdout bytes.Buffer

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = path
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf(
				"error executing %s: %s",
				strings.Join(args, " "),
				err)
		}

		// Read each line of output as a path
		result := make([]string, 0, 100)
		scanner := bufio.NewScanner(&stdout)
		for scanner.Scan() {
			result = append(result, scanner.Text())
		}

		// Always use *nix-style paths (for Windows)
		for idx, value := range result {
			result[idx] = filepath.ToSlash(value)
		}

		return result, nil
	}
}

// vcsTrimCmd trims the prefix from the paths returned by another VCSFilesFunc.
// This should be used to wrap another function if the return value is known
// to have full paths rather than relative paths
func vcsTrimCmd(f VCSFilesFunc) VCSFilesFunc {
	return func(path string) ([]string, error) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf(
				"error expanding VCS path: %s", err)
		}

		// Now that we have the root path, get the inner files
		fs, err := f(path)
		if err != nil {
			return nil, err
		}

		// Trim the root path from the files
		result := make([]string, 0, len(fs))
		for _, f := range fs {
			if !strings.HasPrefix(f, absPath) {
				continue
			}

			f, err = filepath.Rel(absPath, f)
			if err != nil {
				return nil, fmt.Errorf(
					"error determining path: %s", err)
			}

			result = append(result, f)
		}

		return result, nil
	}
}

// vcsMetadata returns the metadata for the VCS directory path.
func vcsMetadata(path string) (map[string]string, error) {
	vcs, err := vcsDetect(path)
	if err != nil {
		return nil, fmt.Errorf("error detecting VCS: %s", err)
	}

	if vcs.Metadata != nil {
		return vcs.Metadata(path)
	}

	return nil, nil
}

const ignorableDetachedHeadError = "HEAD is not a symbolic ref"

// gitBranch gets and returns the current git branch for the Git repository
// at the given path. It is assumed that the VCS is git.
func gitBranch(path string) (string, error) {
	var stderr, stdout bytes.Buffer

	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	cmd.Dir = path
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), ignorableDetachedHeadError) {
			return "", nil
		} else {
			return "",
				fmt.Errorf("error getting git branch: %s\nstdout: %s\nstderr: %s",
					err, stdout.String(), stderr.String())
		}
	}

	branch := strings.TrimSpace(stdout.String())

	return branch, nil
}

// gitCommit gets the SHA of the latest commit for the Git repository at the
// given path. It is assumed that the VCS is git.
func gitCommit(path string) (string, error) {
	var stderr, stdout bytes.Buffer

	cmd := exec.Command("git", "log", "-n1", "--pretty=format:%H")
	cmd.Dir = path
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error getting git commit: %s\nstdout: %s\nstderr: %s",
			err, stdout.String(), stderr.String())
	}

	commit := strings.TrimSpace(stdout.String())

	return commit, nil
}

// gitRemotes gets and returns a map of all remotes for the Git repository. The
// map key is the name of the remote of the format "remote.NAME" and the value
// is the endpoint for the remote. It is assumed that the VCS is git.
func gitRemotes(path string) (map[string]string, error) {
	var stderr, stdout bytes.Buffer

	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = path
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error getting git remotes: %s\nstdout: %s\nstderr: %s",
			err, stdout.String(), stderr.String())
	}

	// Read each line of output as a remote
	result := make(map[string]string)
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := scanner.Text()
		split := strings.Split(line, "\t")

		if len(split) < 2 {
			return nil, fmt.Errorf("invalid response from git remote: %s", stdout.String())
		}

		remote := fmt.Sprintf("remote.%s", strings.TrimSpace(split[0]))
		if _, ok := result[remote]; !ok {
			// https://github.com/foo/bar.git (fetch) #=> https://github.com/foo/bar.git
			urlSplit := strings.Split(split[1], " ")
			result[remote] = strings.TrimSpace(urlSplit[0])
		}
	}

	return result, nil
}

// gitPreflight is the pre-flight command that runs for Git-based VCSs
func gitPreflight(path string) error {
	var stderr, stdout bytes.Buffer

	cmd := exec.Command("git", "--version")
	cmd.Dir = path
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error getting git version: %s\nstdout: %s\nstderr: %s",
			err, stdout.String(), stderr.String())
	}

	// Check if the output is valid
	output := strings.Split(strings.TrimSpace(stdout.String()), " ")
	if len(output) < 1 {
		log.Printf("[WARN] could not extract version output from Git")
		return nil
	}

	// Parse the version
	gitv, err := version.NewVersion(output[len(output)-1])
	if err != nil {
		log.Printf("[WARN] could not parse version output from Git")
		return nil
	}

	constraint, err := version.NewConstraint("> 1.8")
	if err != nil {
		log.Printf("[WARN] could not create version constraint to check")
		return nil
	}
	if !constraint.Check(gitv) {
		return fmt.Errorf("git version (%s) is too old, please upgrade", gitv.String())
	}

	return nil
}

// gitMetadata is the function to parse and return Git metadata
func gitMetadata(path string) (map[string]string, error) {
	// Future-self note: Git is NOT threadsafe, so we cannot run these
	// operations in go routines or else you're going to have a really really
	// bad day and Panda.State == "Sad" :(

	branch, err := gitBranch(path)
	if err != nil {
		return nil, err
	}

	commit, err := gitCommit(path)
	if err != nil {
		return nil, err
	}

	remotes, err := gitRemotes(path)
	if err != nil {
		return nil, err
	}

	// Make the return result (we already know the size)
	result := make(map[string]string, 2+len(remotes))

	result["branch"] = branch
	result["commit"] = commit
	for remote, value := range remotes {
		result[remote] = value
	}

	return result, nil
}
