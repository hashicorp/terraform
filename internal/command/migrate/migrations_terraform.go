// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package migrate

import (
	"fmt"
	"regexp"
)

func terraformMigrations() []Migration {
	return []Migration{
		{
			Namespace:   "terraform",
			Provider:    "terraform",
			Name:        "v1.x-to-v2.x",
			Description: "Migrates Terraform core configuration patterns from v1.x to v2.x.",
			SubMigrations: []SubMigration{
				{
					Name:        "required-providers-map",
					Description: "Converts old-style required_providers string syntax to the new object syntax with source and version.",
					Apply:       applyRequiredProvidersMap,
				},
				{
					Name:        "backend-to-cloud",
					Description: "Converts backend \"remote\" blocks to the new cloud block syntax.",
					Apply:       applyBackendToCloud,
				},
			},
		},
	}
}

// oldStyleProviderLineRe matches direct old-style provider entries:
//
//	aws = "~> 3.0"
//
// It requires the value to NOT start with { (which would be the new object style).
// It also requires the key to be a simple identifier (not "source" or "version").
var oldStyleProviderLineRe = regexp.MustCompile(`(?m)^(\s+)(\w+)\s*=\s*"([^"]+)"\s*$`)

func applyRequiredProvidersMap(_ string, src []byte) ([]byte, error) {
	s := string(src)

	// Find the required_providers block boundaries
	_, _, start, end, ok := extractNestedBlock(s, "required_providers")
	if !ok {
		return src, nil
	}

	// Extract just the block region from the source
	block := s[start:end]

	// Process line by line within the block, tracking brace depth to only
	// replace old-style entries at the top level of the required_providers block.
	lines := splitLines(block)
	depth := 0
	changed := false
	var newLines []string

	for _, line := range lines {
		// Count braces before attempting match
		for _, ch := range line {
			switch ch {
			case '{':
				depth++
			case '}':
				depth--
			}
		}

		// depth==1 means we're at the top level inside required_providers { }
		if depth == 1 {
			m := oldStyleProviderLineRe.FindStringSubmatch(line)
			if m != nil {
				indent := m[1]
				providerName := m[2]
				version := m[3]
				replacement := fmt.Sprintf("%s%s = {\n%s  source  = \"hashicorp/%s\"\n%s  version = %q\n%s}",
					indent, providerName, indent, providerName, indent, version, indent)
				newLines = append(newLines, replacement)
				changed = true
				continue
			}
		}

		newLines = append(newLines, line)
	}

	if !changed {
		return src, nil
	}

	newBlock := joinLines(newLines)
	// joinLines prepends \n to each line, so the result starts with \n.
	// Remove the leading \n since we're replacing in-place.
	if len(newBlock) > 0 && newBlock[0] == '\n' {
		newBlock = newBlock[1:]
	}

	result := s[:start] + newBlock + s[end:]
	return []byte(result), nil
}

// splitLines splits a string into lines, preserving the content but not the
// trailing newline characters.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func joinLines(lines []string) string {
	result := ""
	for _, line := range lines {
		result += "\n" + line
	}
	return result
}


var backendRemoteBlockRe = regexp.MustCompile(`(?s)([ \t]*)backend\s+"remote"\s*\{`)

func applyBackendToCloud(_ string, src []byte) ([]byte, error) {
	s := string(src)

	m := backendRemoteBlockRe.FindStringSubmatchIndex(s)
	if m == nil {
		return src, nil
	}

	// m[2]:m[3] is the indent capture group
	indent := s[m[2]:m[3]]

	// Find the opening brace position
	bracePos := m[0]
	for bracePos < m[1] && s[bracePos] != '{' {
		bracePos++
	}

	// Count braces to find the matching close
	depth := 1
	i := bracePos + 1
	for i < len(s) && depth > 0 {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
		}
		i++
	}
	if depth != 0 {
		return src, nil
	}

	// Extract the inner content (between braces)
	innerContent := s[bracePos+1 : i-1]

	// Build replacement
	replacement := indent + "cloud {" + innerContent + "}"
	result := s[:m[0]] + replacement + s[i:]

	return []byte(result), nil
}
