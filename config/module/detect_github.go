package module

import (
	"fmt"
	"net/url"
	"strings"
)

// GitHubDetector implements Detector to detect GitHub URLs and turn
// them into URLs that the Git Getter can understand.
type GitHubDetector struct{}

func (d *GitHubDetector) Detect(src, _ string) (string, bool, error) {
	if len(src) == 0 {
		return "", false, nil
	}

	if strings.HasPrefix(src, "github.com/") {
		return d.detectHTTP(src)
	} else if strings.HasPrefix(src, "git@github.com:") {
		return d.detectSSH(src)
	}

	return "", false, nil
}

func (d *GitHubDetector) detectHTTP(src string) (string, bool, error) {
	urlStr := fmt.Sprintf("https://%s", src)
	url, err := url.Parse(urlStr)
	if err != nil {
		return "", true, fmt.Errorf("error parsing GitHub URL: %s", err)
	}

	if !strings.HasSuffix(url.Path, ".git") {
		url.Path += ".git"
	}

	return "git::" + url.String(), true, nil
}

func (d *GitHubDetector) detectSSH(src string) (string, bool, error) {
	idx := strings.Index(src, ":")
	qidx := strings.Index(src, "?")
	if qidx == -1 {
		qidx = len(src)
	}

	var u url.URL
	u.Scheme = "ssh"
	u.User = url.User("git")
	u.Host = "github.com"
	u.Path = src[idx+1:qidx]
	if qidx < len(src) {
		q, err := url.ParseQuery(src[qidx+1:])
		if err != nil {
			return "", true, fmt.Errorf("error parsing GitHub SSH URL: %s", err)
		}

		u.RawQuery = q.Encode()
	}

	return "git::"+u.String(), true, nil
}
