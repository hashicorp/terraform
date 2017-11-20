package regsrc

import (
	"regexp"
	"strings"

	"github.com/hashicorp/terraform/svchost"
)

var (
	// InvalidHostString is a placeholder returned when a raw host can't be
	// converted by IDNA spec. It will never be returned for any host for which
	// Valid() is true.
	InvalidHostString = "<invalid host>"

	// urlLabelEndSubRe is a sub-expression that matches any character that's
	// allowed at the start or end of a URL label according to RFC1123.
	urlLabelEndSubRe = "[0-9A-Za-z]"

	// urlLabelEndSubRe is a sub-expression that matches any character that's
	// allowed at in a non-start or end of a URL label according to RFC1123.
	urlLabelMidSubRe = "[0-9A-Za-z-]"

	// urlLabelUnicodeSubRe is a sub-expression that matches any non-ascii char
	// in an IDN (Unicode) display URL. It's not strict - there are only ~15k
	// valid Unicode points in IDN RFC (some with conditions). We are just going
	// with being liberal with matching and then erroring if we fail to convert
	// to punycode later (which validates chars fully). This at least ensures
	// ascii chars dissalowed by the RC1123 parts above don't become legal
	// again.
	urlLabelUnicodeSubRe = "[^[:ascii:]]"

	// hostLabelSubRe is the sub-expression that matches a valid hostname label.
	// It does not anchor the start or end so it can be composed into more
	// complex RegExps below. Note that for sanity we don't handle disallowing
	// raw punycode in this regexp (esp. since re2 doesn't support negative
	// lookbehind, but we can capture it's presence here to check later).
	hostLabelSubRe = "" +
		// Match valid initial char, or unicode char
		"(?:" + urlLabelEndSubRe + "|" + urlLabelUnicodeSubRe + ")" +
		// Optionally, match 0 to 61 valid URL or Unicode chars,
		// followed by one valid end char or unicode char
		"(?:" +
		"(?:" + urlLabelMidSubRe + "|" + urlLabelUnicodeSubRe + "){0,61}" +
		"(?:" + urlLabelEndSubRe + "|" + urlLabelUnicodeSubRe + ")" +
		")?"

	// hostSubRe is the sub-expression that matches a valid host prefix.
	// Allows custom port.
	hostSubRe = hostLabelSubRe + "(?:\\." + hostLabelSubRe + ")+(?::\\d+)?"

	// hostRe is a regexp that matches a valid host prefix. Additional
	// validation of unicode strings is needed for matches.
	hostRe = regexp.MustCompile("^" + hostSubRe + "$")
)

// FriendlyHost describes a registry instance identified in source strings by a
// simple bare hostname like registry.terraform.io.
type FriendlyHost struct {
	Raw string
}

func NewFriendlyHost(host string) *FriendlyHost {
	return &FriendlyHost{Raw: host}
}

// ParseFriendlyHost attempts to parse a valid "friendly host" prefix from the
// given string. If no valid prefix is found, host will be nil and rest will
// contain the full source string. The host prefix must terminate at the end of
// the input or at the first / character. If one or more characters exist after
// the first /, they will be returned as rest (without the / delimiter).
// Hostnames containing punycode WILL be parsed successfully since they may have
// come from an internal normalized source string, however should be considered
// invalid if the string came from a user directly. This must be checked
// explicitly for user-input strings by calling Valid() on the
// returned host.
func ParseFriendlyHost(source string) (host *FriendlyHost, rest string) {
	parts := strings.SplitN(source, "/", 2)

	if hostRe.MatchString(parts[0]) {
		host = &FriendlyHost{Raw: parts[0]}
		if len(parts) == 2 {
			rest = parts[1]
		}
		return
	}

	// No match, return whole string as rest along with nil host
	rest = source
	return
}

// Valid returns whether the host prefix is considered valid in any case.
// Example of invalid prefixes might include ones that don't conform to the host
// name specifications. Not that IDN prefixes containing punycode are not valid
// input which we expect to always be in user-input or normalised display form.
func (h *FriendlyHost) Valid() bool {
	return svchost.IsValid(h.Raw)
}

// Display returns the host formatted for display to the user in CLI or web
// output.
func (h *FriendlyHost) Display() string {
	return svchost.ForDisplay(h.Raw)
}

// Normalized returns the host formatted for internal reference or comparison.
func (h *FriendlyHost) Normalized() string {
	host, err := svchost.ForComparison(h.Raw)
	if err != nil {
		return InvalidHostString
	}
	return string(host)
}

// String returns the host formatted as the user originally typed it assuming it
// was parsed from user input.
func (h *FriendlyHost) String() string {
	return h.Raw
}

// Equal compares the FriendlyHost against another instance taking normalization
// into account. Invalid hosts cannot be compared and will always return false.
func (h *FriendlyHost) Equal(other *FriendlyHost) bool {
	if other == nil {
		return false
	}

	return h.Normalized() == other.Normalized()
}
