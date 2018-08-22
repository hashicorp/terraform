package utils

import (
	"net/url"
	"regexp"
	"strings"
)

// BaseEndpoint will return a URL without the /vX.Y
// portion of the URL.
func BaseEndpoint(endpoint string) (string, error) {
	var base string

	u, err := url.Parse(endpoint)
	if err != nil {
		return base, err
	}

	u.RawQuery, u.Fragment = "", ""

	versionRe := regexp.MustCompile("v[0-9.]+/?")
	if version := versionRe.FindString(u.Path); version != "" {
		base = strings.Replace(u.String(), version, "", -1)
	} else {
		base = u.String()
	}

	return base, nil
}
