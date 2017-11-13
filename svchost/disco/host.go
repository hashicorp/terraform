package disco

import (
	"net/url"
)

type Host struct {
	discoURL *url.URL
	services map[string]interface{}
}

// ServiceURL returns the URL associated with the given service identifier,
// which should be of the form "servicename.vN".
//
// A non-nil result is always an absolute URL with a scheme of either https
// or http.
//
// If the requested service is not supported by the host, this method returns
// a nil URL.
//
// If the discovery document entry for the given service is invalid (not a URL),
// it is treated as absent, also returning a nil URL.
func (h Host) ServiceURL(id string) *url.URL {
	if h.services == nil {
		return nil // no services supported for an empty Host
	}

	urlStr, ok := h.services[id].(string)
	if !ok {
		return nil
	}

	ret, err := url.Parse(urlStr)
	if err != nil {
		return nil
	}
	if !ret.IsAbs() {
		ret = h.discoURL.ResolveReference(ret) // make absolute using our discovery doc URL
	}
	if ret.Scheme != "https" && ret.Scheme != "http" {
		return nil
	}
	if ret.User != nil {
		// embedded username/password information is not permitted; credentials
		// are handled out of band.
		return nil
	}
	ret.Fragment = "" // fragment part is irrelevant, since we're not a browser

	return h.discoURL.ResolveReference(ret)
}
