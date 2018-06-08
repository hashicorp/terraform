// Package svchost deals with the representations of the so-called "friendly
// hostnames" that we use to represent systems that provide Terraform-native
// remote services, such as module registry, remote operations, etc.
//
// Friendly hostnames are specified such that, as much as possible, they
// are consistent with how web browsers think of hostnames, so that users
// can bring their intuitions about how hostnames behave when they access
// a Terraform Enterprise instance's web UI (or indeed any other website)
// and have this behave in a similar way.
package svchost

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/net/idna"
)

// Hostname is specialized name for string that indicates that the string
// has been converted to (or was already in) the storage and comparison form.
//
// Hostname values are not suitable for display in the user-interface. Use
// the ForDisplay method to obtain a form suitable for display in the UI.
//
// Unlike user-supplied hostnames, strings of type Hostname (assuming they
// were constructed by a function within this package) can be compared for
// equality using the standard Go == operator.
type Hostname string

// acePrefix is the ASCII Compatible Encoding prefix, used to indicate that
// a domain name label is in "punycode" form.
const acePrefix = "xn--"

// displayProfile is a very liberal idna profile that we use to do
// normalization for display without imposing validation rules.
var displayProfile = idna.New(
	idna.MapForLookup(),
	idna.Transitional(true),
)

// ForDisplay takes a user-specified hostname and returns a normalized form of
// it suitable for display in the UI.
//
// If the input is so invalid that no normalization can be performed then
// this will return the input, assuming that the caller still wants to
// display _something_. This function is, however, more tolerant than the
// other functions in this package and will make a best effort to prepare
// _any_ given hostname for display.
//
// For validation, use either IsValid (for explicit validation) or
// ForComparison (which implicitly validates, returning an error if invalid).
func ForDisplay(given string) string {
	var portPortion string
	if colonPos := strings.Index(given, ":"); colonPos != -1 {
		given, portPortion = given[:colonPos], given[colonPos:]
	}
	portPortion, _ = normalizePortPortion(portPortion)

	ascii, err := displayProfile.ToASCII(given)
	if err != nil {
		return given + portPortion
	}
	display, err := displayProfile.ToUnicode(ascii)
	if err != nil {
		return given + portPortion
	}
	return display + portPortion
}

// IsValid returns true if the given user-specified hostname is a valid
// service hostname.
//
// Validity is determined by complying with the RFC 5891 requirements for
// names that are valid for domain lookup (section 5), with the additional
// requirement that user-supplied forms must not _already_ contain
// Punycode segments.
func IsValid(given string) bool {
	_, err := ForComparison(given)
	return err == nil
}

// ForComparison takes a user-specified hostname and returns a normalized
// form of it suitable for storage and comparison. The result is not suitable
// for display to end-users because it uses Punycode to represent non-ASCII
// characters, and this form is unreadable for non-ASCII-speaking humans.
//
// The result is typed as Hostname -- a specialized name for string -- so that
// other APIs can make it clear within the type system whether they expect a
// user-specified or display-form hostname or a value already normalized for
// comparison.
//
// The returned Hostname is not valid if the returned error is non-nil.
func ForComparison(given string) (Hostname, error) {
	var portPortion string
	if colonPos := strings.Index(given, ":"); colonPos != -1 {
		given, portPortion = given[:colonPos], given[colonPos:]
	}

	var err error
	portPortion, err = normalizePortPortion(portPortion)
	if err != nil {
		return Hostname(""), err
	}

	if given == "" {
		return Hostname(""), fmt.Errorf("empty string is not a valid hostname")
	}

	// First we'll apply our additional constraint that Punycode must not
	// be given directly by the user. This is not an IDN specification
	// requirement, but we prohibit it to force users to use human-readable
	// hostname forms within Terraform configuration.
	labels := labelIter{orig: given}
	for ; !labels.done(); labels.next() {
		label := labels.label()
		if label == "" {
			return Hostname(""), fmt.Errorf(
				"hostname contains empty label (two consecutive periods)",
			)
		}
		if strings.HasPrefix(label, acePrefix) {
			return Hostname(""), fmt.Errorf(
				"hostname label %q specified in punycode format; service hostnames must be given in unicode",
				label,
			)
		}
	}

	result, err := idna.Lookup.ToASCII(given)
	if err != nil {
		return Hostname(""), err
	}
	return Hostname(result + portPortion), nil
}

// ForDisplay returns a version of the receiver that is appropriate for display
// in the UI. This includes converting any punycode labels to their
// corresponding Unicode characters.
//
// A round-trip through ForComparison and this ForDisplay method does not
// guarantee the same result as calling this package's top-level ForDisplay
// function, since a round-trip through the Hostname type implies stricter
// handling than we do when doing basic display-only processing.
func (h Hostname) ForDisplay() string {
	given := string(h)
	var portPortion string
	if colonPos := strings.Index(given, ":"); colonPos != -1 {
		given, portPortion = given[:colonPos], given[colonPos:]
	}
	// We don't normalize the port portion here because we assume it's
	// already been normalized on the way in.

	result, err := idna.Lookup.ToUnicode(given)
	if err != nil {
		// Should never happen, since type Hostname indicates that a string
		// passed through our validation rules.
		panic(fmt.Errorf("ForDisplay called on invalid Hostname: %s", err))
	}
	return result + portPortion
}

func (h Hostname) String() string {
	return string(h)
}

func (h Hostname) GoString() string {
	return fmt.Sprintf("svchost.Hostname(%q)", string(h))
}

// normalizePortPortion attempts to normalize the "port portion" of a hostname,
// which begins with the first colon in the hostname and should be followed
// by a string of decimal digits.
//
// If the port portion is valid, a normalized version of it is returned along
// with a nil error.
//
// If the port portion is invalid, the input string is returned verbatim along
// with a non-nil error.
//
// An empty string is a valid port portion representing the absense of a port.
// If non-empty, the first character must be a colon.
func normalizePortPortion(s string) (string, error) {
	if s == "" {
		return s, nil
	}

	if s[0] != ':' {
		// should never happen, since caller tends to guarantee the presence
		// of a colon due to how it's extracted from the string.
		return s, errors.New("port portion is missing its initial colon")
	}

	numStr := s[1:]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return s, errors.New("port portion contains non-digit characters")
	}
	if num == 443 {
		return "", nil // ":443" is the default
	}
	if num > 65535 {
		return s, errors.New("port number is greater than 65535")
	}
	return fmt.Sprintf(":%d", num), nil
}
