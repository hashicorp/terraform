// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduleaddrs

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// detectGit translates Git SSH URLs into normal-shaped URLs.
func detectGit(src string) (string, bool, error) {
	if len(src) == 0 {
		return "", false, nil
	}

	u, err := detectSSH(src)
	if err != nil {
		return "", true, err
	}
	if u == nil {
		return "", false, nil
	}

	// We require the username to be "git" to assume that this is a Git URL
	if u.User.Username() != "git" {
		return "", false, nil
	}

	return "git::" + u.String(), true, nil
}

// detectGitHub detects shorthand schemeless references to github.com and
// translates them into git HTTP source addresses.
func detectGitHub(src string) (string, bool, error) {
	if len(src) == 0 {
		return "", false, nil
	}

	if strings.HasPrefix(src, "github.com/") {
		src, rawQuery, _ := strings.Cut(src, "?")

		parts := strings.Split(src, "/")
		if len(parts) < 3 {
			return "", false, fmt.Errorf(
				"GitHub URLs should be github.com/username/repo")
		}

		urlStr := fmt.Sprintf("https://%s", strings.Join(parts[:3], "/"))
		url, err := url.Parse(urlStr)
		if err != nil {
			return "", true, fmt.Errorf("error parsing GitHub URL: %s", err)
		}
		url.RawQuery = rawQuery

		if !strings.HasSuffix(url.Path, ".git") {
			url.Path += ".git"
		}

		if len(parts) > 3 {
			url.Path += "//" + strings.Join(parts[3:], "/")
		}

		return "git::" + url.String(), true, nil
	}

	return "", false, nil
}

// detectBitBucket detects shorthand schemeless references to bitbucket.org and
// translates them into git HTTP source addresses.
func detectBitBucket(src string) (string, bool, error) {
	if len(src) == 0 {
		return "", false, nil
	}

	if strings.HasPrefix(src, "bitbucket.org/") {
		u, err := url.Parse("https://" + src)
		if err != nil {
			return "", true, fmt.Errorf("error parsing BitBucket URL: %s", err)
		}

		// NOTE: A long, long time ago bitbucket.org repositories could
		// potentially be either Git or Mercurial repositories and we would've
		// needed to make an API call here to know which to generate.
		//
		// Thankfully BitBucket now only supports Git, and so we can just
		// assume all bitbucket.org strings are trying to refer to Git
		// repositories.

		if !strings.HasSuffix(u.Path, ".git") {
			u.Path += ".git"
		}

		return "git::" + u.String(), true, nil
	}

	return "", false, nil
}

// sshPattern matches SCP-like SSH patterns (user@host:path)
var sshPattern = regexp.MustCompile("^(?:([^@]+)@)?([^:]+):/?(.+)$")

// detectSSH determines if the src string matches an SSH-like URL and
// converts it into a net.URL. This returns nil if the string doesn't match
// the SSH pattern.
func detectSSH(src string) (*url.URL, error) {
	matched := sshPattern.FindStringSubmatch(src)
	if matched == nil {
		return nil, nil
	}

	user := matched[1]
	host := matched[2]
	path := matched[3]
	qidx := strings.Index(path, "?")
	if qidx == -1 {
		qidx = len(path)
	}

	var u url.URL
	u.Scheme = "ssh"
	u.User = url.User(user)
	u.Host = host
	u.Path = path[0:qidx]
	if qidx < len(path) {
		q, err := url.ParseQuery(path[qidx+1:])
		if err != nil {
			return nil, fmt.Errorf("error parsing Git SSH URL: %s", err)
		}
		u.RawQuery = q.Encode()
	}

	return &u, nil
}
