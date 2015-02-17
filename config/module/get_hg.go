package module

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"

	urlhelper "github.com/hashicorp/terraform/helper/url"
)

// HgGetter is a Getter implementation that will download a module from
// a Mercurial repository.
type HgGetter struct{}

func (g *HgGetter) Get(dst string, u *url.URL) error {
	if _, err := exec.LookPath("hg"); err != nil {
		return fmt.Errorf("hg must be available and on the PATH")
	}

	newURL, err := urlhelper.Parse(u.String())
	if err != nil {
		return err
	}
	if fixWindowsDrivePath(newURL) {
		// See valid file path form on http://www.selenic.com/hg/help/urls
		newURL.Path = fmt.Sprintf("/%s", newURL.Path)
	}

	// Extract some query parameters we use
	var rev string
	q := newURL.Query()
	if len(q) > 0 {
		rev = q.Get("rev")
		q.Del("rev")

		newURL.RawQuery = q.Encode()
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err != nil {
		if err := g.clone(dst, newURL); err != nil {
			return err
		}
	}

	if err := g.pull(dst, newURL); err != nil {
		return err
	}

	return g.update(dst, newURL, rev)
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

func fixWindowsDrivePath(u *url.URL) bool {
	// hg assumes a file:/// prefix for Windows drive letter file paths.
	// (e.g. file:///c:/foo/bar)
	// If the URL Path does not begin with a '/' character, the resulting URL
	// path will have a file:// prefix. (e.g. file://c:/foo/bar)
	// See http://www.selenic.com/hg/help/urls and the examples listed in
	// http://selenic.com/repo/hg-stable/file/1265a3a71d75/mercurial/util.py#l1936
	return runtime.GOOS == "windows" && u.Scheme == "file" &&
		len(u.Path) > 1 && u.Path[0] != '/' && u.Path[1] == ':'
}
