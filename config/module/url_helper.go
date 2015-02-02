package module

import (
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
)

func urlParse(rawURL string) (*url.URL, error) {
	if runtime.GOOS == "windows" {
		// Make sure we're using "/" on Windows. URLs are "/"-based.
		rawURL = filepath.ToSlash(rawURL)
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	if runtime.GOOS != "windows" {
		return u, err
	}

	if len(rawURL) > 1 && rawURL[1] == ':' {
		// Assume we're dealing with a drive letter file path on Windows.
		// We need to adjust the URL Path for drive letter file paths
		// because url.Parse("c:/users/user") yields URL Scheme = "c"
		// and URL path = "/users/user".
		u.Path = fmt.Sprintf("%s:%s", u.Scheme, u.Path)
		u.Scheme = ""
	}

	// Remove leading slash for absolute file paths on Windows.
	// For example, url.Parse yields u.Path = "/C:/Users/user" for
	// rawURL = "file:///C:/Users/user", which is an incorrect syntax.
	if len(u.Path) > 2 && u.Path[0] == '/' && u.Path[2] == ':' {
		u.Path = u.Path[1:]
	}

	return u, err
}

func fmtFileURL(path string) string {
	if runtime.GOOS == "windows" {
		// Make sure we're using "/" on Windows. URLs are "/"-based.
		path = filepath.ToSlash(path)
	}

	// Make sure that we don't start with "/" since we add that below.
	if path[0] == '/' {
		path = path[1:]
	}

	return fmt.Sprintf("file:///%s", path)
}
