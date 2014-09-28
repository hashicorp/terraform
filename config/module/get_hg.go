package module

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
)

// HgGetter is a Getter implementation that will download a module from
// a Mercurial repository.
type HgGetter struct{}

func (g *HgGetter) Get(dst string, u *url.URL) error {
	if _, err := exec.LookPath("hg"); err != nil {
		return fmt.Errorf("hg must be available and on the PATH")
	}

	// Extract some query parameters we use
	var rev string
	q := u.Query()
	if len(q) > 0 {
		rev = q.Get("rev")
		q.Del("rev")

		// Copy the URL
		var newU url.URL = *u
		u = &newU
		u.RawQuery = q.Encode()
	}

	_, err := os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err != nil {
		if err := g.clone(dst, u); err != nil {
			return err
		}
	}

	if err := g.pull(dst, u); err != nil {
		return err
	}

	return g.update(dst, u, rev)
}

func (g *HgGetter) clone(dst string, u *url.URL) error {
	cmd := exec.Command("hg", "clone", "-U", u.String(), dst)
	return getRunCommand(cmd)
}

func (g *HgGetter) pull(dst string, u *url.URL) error {
	cmd := exec.Command("hg", "pull")
	cmd.Dir = dst
	return getRunCommand(cmd)
}

func (g *HgGetter) update(dst string, u *url.URL, rev string) error {
	args := []string{"update"}
	if rev != "" {
		args = append(args, rev)
	}

	cmd := exec.Command("hg", args...)
	cmd.Dir = dst
	return getRunCommand(cmd)
}
