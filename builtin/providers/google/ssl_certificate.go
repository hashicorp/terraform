package google

import (
	"fmt"
	"regexp"
)

const (
	canonicalizeCertUrlUrlPrefix       = "https://www.googleapis.com/compute/v1/"
	canonicalizeCertUrlPartialUrlRegex = "projects/[a-z](?:[-a-z0-9]*[a-z0-9])?/global/sslCertificates/[a-z](?:[-a-z0-9]*[a-z0-9])?"
)

var (
	canonicalizeCertUrlFull    = regexp.MustCompile(fmt.Sprintf("^%s%s$", canonicalizeCertUrlUrlPrefix, canonicalizeCertUrlPartialUrlRegex))
	canonicalizeCertUrlPartial = regexp.MustCompile(fmt.Sprintf("^%s$", canonicalizeCertUrlPartialUrlRegex))
)

func canonicalizeCertUrl(url string) (string, error) {
	switch {
	case canonicalizeCertUrlFull.MatchString(url):
		return url, nil
	case canonicalizeCertUrlPartial.MatchString(url):
		return canonicalizeCertUrlUrlPrefix + url, nil
	}
	return "", fmt.Errorf("Invalid URL '%s'", url)
}
