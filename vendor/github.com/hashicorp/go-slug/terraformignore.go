package slug

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/scanner"
)

func parseIgnoreFile(rootPath string) []rule {
	// Look for .terraformignore at our root path/src
	file, err := os.Open(filepath.Join(rootPath, ".terraformignore"))
	defer file.Close()

	// If there's any kind of file error, punt and use the default ignore patterns
	if err != nil {
		// Only show the error debug if an error *other* than IsNotExist
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error reading .terraformignore, default exclusions will apply: %v \n", err)
		}
		return defaultExclusions
	}
	return readRules(file)
}

func readRules(input io.Reader) []rule {
	rules := defaultExclusions
	scanner := bufio.NewScanner(input)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		pattern := scanner.Text()
		// Ignore blank lines
		if len(pattern) == 0 {
			continue
		}
		// Trim spaces
		pattern = strings.TrimSpace(pattern)
		// Ignore comments
		if pattern[0] == '#' {
			continue
		}
		// New rule structure
		rule := rule{}
		// Exclusions
		if pattern[0] == '!' {
			rule.excluded = true
			pattern = pattern[1:]
		}
		// If it is a directory, add ** so we catch descendants
		if pattern[len(pattern)-1] == os.PathSeparator {
			pattern = pattern + "**"
		}
		// If it starts with /, it is absolute
		if pattern[0] == os.PathSeparator {
			pattern = pattern[1:]
		} else {
			// Otherwise prepend **/
			pattern = "**" + string(os.PathSeparator) + pattern
		}
		rule.val = pattern
		rule.dirs = strings.Split(pattern, string(os.PathSeparator))
		rules = append(rules, rule)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading .terraformignore, default exclusions will apply: %v \n", err)
		return defaultExclusions
	}
	return rules
}

func matchIgnoreRule(path string, rules []rule) bool {
	matched := false
	path = filepath.FromSlash(path)
	for _, rule := range rules {
		match, _ := rule.match(path)

		if match {
			matched = !rule.excluded
		}
	}

	if matched {
		debug(true, path, "Skipping excluded path:", path)
	}

	return matched
}

type rule struct {
	val      string         // the value of the rule itself
	excluded bool           // ! is present, an exclusion rule
	dirs     []string       // directories of the rule
	regex    *regexp.Regexp // regular expression to match for the rule
}

func (r *rule) match(path string) (bool, error) {
	if r.regex == nil {
		if err := r.compile(); err != nil {
			return false, filepath.ErrBadPattern
		}
	}

	b := r.regex.MatchString(path)
	debug(false, path, path, r.regex, b)
	return b, nil
}

func (r *rule) compile() error {
	regStr := "^"
	pattern := r.val
	// Go through the pattern and convert it to a regexp.
	// Use a scanner to support utf-8 chars.
	var scan scanner.Scanner
	scan.Init(strings.NewReader(pattern))

	sl := string(os.PathSeparator)
	escSL := sl
	if sl == `\` {
		escSL += `\`
	}

	for scan.Peek() != scanner.EOF {
		ch := scan.Next()
		if ch == '*' {
			if scan.Peek() == '*' {
				// is some flavor of "**"
				scan.Next()

				// Treat **/ as ** so eat the "/"
				if string(scan.Peek()) == sl {
					scan.Next()
				}

				if scan.Peek() == scanner.EOF {
					// is "**EOF" - to align with .gitignore just accept all
					regStr += ".*"
				} else {
					// is "**"
					// Note that this allows for any # of /'s (even 0) because
					// the .* will eat everything, even /'s
					regStr += "(.*" + escSL + ")?"
				}
			} else {
				// is "*" so map it to anything but "/"
				regStr += "[^" + escSL + "]*"
			}
		} else if ch == '?' {
			// "?" is any char except "/"
			regStr += "[^" + escSL + "]"
		} else if ch == '.' || ch == '$' {
			// Escape some regexp special chars that have no meaning
			// in golang's filepath.Match
			regStr += `\` + string(ch)
		} else if ch == '\\' {
			// escape next char. Note that a trailing \ in the pattern
			// will be left alone (but need to escape it)
			if sl == `\` {
				// On windows map "\" to "\\", meaning an escaped backslash,
				// and then just continue because filepath.Match on
				// Windows doesn't allow escaping at all
				regStr += escSL
				continue
			}
			if scan.Peek() != scanner.EOF {
				regStr += `\` + string(scan.Next())
			} else {
				regStr += `\`
			}
		} else {
			regStr += string(ch)
		}
	}

	regStr += "$"
	re, err := regexp.Compile(regStr)
	if err != nil {
		return err
	}

	r.regex = re
	return nil
}

/*
	Default rules as they would appear in .terraformignore:
	.git/
	.terraform/
	!.terraform/modules/
*/

var defaultExclusions = []rule{
	{
		val:      strings.Join([]string{"**", ".git", "**"}, string(os.PathSeparator)),
		excluded: false,
	},
	{
		val:      strings.Join([]string{"**", ".terraform", "**"}, string(os.PathSeparator)),
		excluded: false,
	},
	{
		val:      strings.Join([]string{"**", ".terraform", "modules", "**"}, string(os.PathSeparator)),
		excluded: true,
	},
}

func debug(printAll bool, path string, message ...interface{}) {
	logLevel := os.Getenv("TF_IGNORE") == "trace"
	debugPath := os.Getenv("TF_IGNORE_DEBUG")
	isPath := debugPath != ""
	if isPath {
		isPath = strings.Contains(path, debugPath)
	}

	if logLevel {
		if printAll || isPath {
			fmt.Println(message...)
		}
	}
}
