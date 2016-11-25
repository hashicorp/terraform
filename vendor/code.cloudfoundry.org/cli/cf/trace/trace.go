package trace

import (
	"fmt"
	"regexp"

	. "code.cloudfoundry.org/cli/cf/i18n"
)

var LoggingToStdout bool

func Sanitize(input string) string {
	re := regexp.MustCompile(`(?m)^Authorization: .*`)
	sanitized := re.ReplaceAllString(input, "Authorization: "+PrivateDataPlaceholder())

	re = regexp.MustCompile(`password=[^&]*&`)
	sanitized = re.ReplaceAllString(sanitized, "password="+PrivateDataPlaceholder()+"&")

	sanitized = sanitizeJSON("token", sanitized)
	sanitized = sanitizeJSON("password", sanitized)

	return sanitized
}

func sanitizeJSON(propertySubstring string, json string) string {
	regex := regexp.MustCompile(fmt.Sprintf(`(?i)"([^"]*%s[^"]*)":\s*"[^\,]*"`, propertySubstring))
	return regex.ReplaceAllString(json, fmt.Sprintf(`"$1":"%s"`, PrivateDataPlaceholder()))
}

func PrivateDataPlaceholder() string {
	return T("[PRIVATE DATA HIDDEN]")
}
