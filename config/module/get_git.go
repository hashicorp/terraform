package module

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
)

// GitGetter is a Getter implementation that will download a module from
// a git repository.
type GitGetter struct{}

func (g *GitGetter) Get(dst string, u *url.URL) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git must be available and on the PATH")
	}

	_, err := os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		return g.update(dst, u)
	}

	return g.clone(dst, u)
}

func (g *GitGetter) clone(dst string, u *url.URL) error {
	cmd := exec.Command("git", "clone", u.String(), dst)
	return getRunCommand(cmd)
}

func (g *GitGetter) update(dst string, u *url.URL) error {
	cmd := exec.Command("git", "pull", "--ff-only")
	cmd.Dir = dst
	return getRunCommand(cmd)
}
