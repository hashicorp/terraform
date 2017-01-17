package internal

import "net/url"

// SafeURL removes sensitive information from a URL.
func SafeURL(u *url.URL) string {
	if nil == u {
		return ""
	}
	if "" != u.Opaque {
		// If the URL is opaque, we cannot be sure if it contains
		// sensitive information.
		return ""
	}

	// Omit user, query, and fragment information for security.
	ur := url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   u.Path,
	}
	return ur.String()
}

// SafeURLFromString removes sensitive information from a URL.
func SafeURLFromString(rawurl string) string {
	u, err := url.Parse(rawurl)
	if nil != err {
		return ""
	}
	return SafeURL(u)
}

// HostFromURL returns the URL's host.
func HostFromURL(u *url.URL) string {
	if nil == u {
		return ""
	}
	if "" != u.Opaque {
		return "opaque"
	}
	return u.Host
}
