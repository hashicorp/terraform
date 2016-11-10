package args

import "regexp"

var (
	envRe = regexp.MustCompile(`\${[a-zA-Z0-9_\-\.]+}`)
)

// ReplaceEnv takes an arg and replaces all occurrences of environment variables.
// If the variable is found in the passed map it is replaced, otherwise the
// original string is returned.
func ReplaceEnv(arg string, environments ...map[string]string) string {
	return envRe.ReplaceAllStringFunc(arg, func(arg string) string {
		stripped := arg[2 : len(arg)-1]
		for _, env := range environments {
			if value, ok := env[stripped]; ok {
				return value
			}
		}

		return arg
	})
}
