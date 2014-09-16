package module

import (
	"fmt"
	"net/url"
)

// Detector defines the interface that an invalid URL or a URL with a blank
// scheme is passed through in order to determine if its shorthand for
// something else well-known.
type Detector interface {
	// Detect will detect whether the string matches a known pattern to
	// turn it into a proper URL.
	Detect(string, string) (string, bool, error)
}

// Detectors is the list of detectors that are tried on an invalid URL.
// This is also the order they're tried (index 0 is first).
var Detectors []Detector

func init() {
	Detectors = []Detector{
		new(FileDetector),
	}
}

// Detect turns a source string into another source string if it is
// detected to be of a known pattern.
//
// This is safe to be called with an already valid source string: Detect
// will just return it.
func Detect(src string, pwd string) (string, error) {
	getForce, getSrc := getForcedGetter(src)

	u, err := url.Parse(getSrc)
	if err == nil && u.Scheme != "" {
		// Valid URL
		return src, nil
	}

	for _, d := range Detectors {
		result, ok, err := d.Detect(getSrc, pwd)
		if err != nil {
			return "", err
		}
		if !ok {
			continue
		}

		// Preserve the forced getter if it exists
		if getForce != "" {
			result = fmt.Sprintf("%s::%s", getForce, result)
		}

		return result, nil
	}

	return "", fmt.Errorf("invalid source string: %s", src)
}
