package module

import (
	"fmt"
	"net/url"
)

// Getter defines the interface that schemes must implement to download
// and update modules.
type Getter interface {
	// Get downloads the given URL into the given directory. This always
	// assumes that we're updating and gets the latest version that it can.
	//
	// The directory may already exist (if we're updating). If it is in a
	// format that isn't understood, an error should be returned. Get shouldn't
	// simply nuke the directory.
	Get(string, *url.URL) error
}

// Getters is the mapping of scheme to the Getter implementation that will
// be used to get a dependency.
var Getters map[string]Getter

func init() {
	Getters = map[string]Getter{
		"file": new(FileGetter),
	}
}

// Get downloads the module specified by src into the folder specified by
// dst. If dst already exists, Get will attempt to update it.
//
// src is a URL, whereas dst is always just a file path to a folder. This
// folder doesn't need to exist. It will be created if it doesn't exist.
func Get(dst, src string) error {
	u, err := url.Parse(src)
	if err != nil {
		return err
	}

	g, ok := Getters[u.Scheme]
	if !ok {
		return fmt.Errorf(
			"module download not supported for scheme '%s'", u.Scheme)
	}

	err = g.Get(dst, u)
	if err != nil {
		err = fmt.Errorf("error downloading module '%s': %s", src, err)
	}

	return err
}
