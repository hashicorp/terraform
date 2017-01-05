package appfiles

import (
	"path"
	"strings"

	"code.cloudfoundry.org/cli/utils/glob"
)

//go:generate counterfeiter . CfIgnore

type CfIgnore interface {
	FileShouldBeIgnored(path string) bool
}

func NewCfIgnore(text string) CfIgnore {
	patterns := []ignorePattern{}
	lines := strings.Split(text, "\n")
	lines = append(defaultIgnoreLines, lines...)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		ignore := true
		if strings.HasPrefix(line, "!") {
			line = line[1:]
			ignore = false
		}

		for _, p := range globsForPattern(path.Clean(line)) {
			patterns = append(patterns, ignorePattern{ignore, p})
		}
	}

	return cfIgnore(patterns)
}

func (ignore cfIgnore) FileShouldBeIgnored(path string) bool {
	result := false

	for _, pattern := range ignore {
		if strings.HasPrefix(pattern.glob.String(), "/") && !strings.HasPrefix(path, "/") {
			path = "/" + path
		}

		if pattern.glob.Match(path) {
			result = pattern.exclude
		}
	}

	return result
}

func globsForPattern(pattern string) (globs []glob.Glob) {
	globs = append(globs, glob.MustCompileGlob(pattern))
	globs = append(globs, glob.MustCompileGlob(path.Join(pattern, "*")))
	globs = append(globs, glob.MustCompileGlob(path.Join(pattern, "**", "*")))

	if !strings.HasPrefix(pattern, "/") {
		globs = append(globs, glob.MustCompileGlob(path.Join("**", pattern)))
		globs = append(globs, glob.MustCompileGlob(path.Join("**", pattern, "*")))
		globs = append(globs, glob.MustCompileGlob(path.Join("**", pattern, "**", "*")))
	}

	return
}

type ignorePattern struct {
	exclude bool
	glob    glob.Glob
}

type cfIgnore []ignorePattern

var defaultIgnoreLines = []string{
	".cfignore",
	"/manifest.yml",
	".gitignore",
	".git",
	".hg",
	".svn",
	"_darcs",
	".DS_Store",
}
