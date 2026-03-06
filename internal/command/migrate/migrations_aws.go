// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package migrate

import (
	"fmt"
	"regexp"
	"strings"
)

// extractNestedBlock finds a named block (e.g. "cors_rule { ... }") within src,
// handling nested braces. Returns the full match, the inner content, and the
// start/end byte offsets. ok is false if no match is found.
func extractNestedBlock(src string, blockName string) (fullMatch, innerContent string, start, end int, ok bool) {
	re := regexp.MustCompile(`(?m)^[ \t]*` + regexp.QuoteMeta(blockName) + `\s*\{`)
	loc := re.FindStringIndex(src)
	if loc == nil {
		return "", "", 0, 0, false
	}

	// Find the opening brace
	braceStart := strings.Index(src[loc[0]:], "{") + loc[0]
	depth := 1
	i := braceStart + 1
	for i < len(src) && depth > 0 {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
		}
		i++
	}
	if depth != 0 {
		return "", "", 0, 0, false
	}

	// i is now one past the closing brace
	fullMatch = src[loc[0]:i]
	innerContent = src[braceStart+1 : i-1]
	return fullMatch, innerContent, loc[0], i, true
}

// dedentBlock removes the leading indentation from a block so it can be
// re-indented at a new level. It strips the common leading whitespace from
// all non-empty lines.
func dedentBlock(block string) string {
	lines := strings.Split(block, "\n")
	// Find minimum indentation of non-empty lines
	minIndent := -1
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" {
			continue
		}
		indent := len(line) - len(trimmed)
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent <= 0 {
		return block
	}
	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			result = append(result, "")
		} else if len(line) >= minIndent {
			result = append(result, line[minIndent:])
		} else {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

// indentBlock adds the given prefix to every non-empty line.
func indentBlock(block, prefix string) string {
	lines := strings.Split(block, "\n")
	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			result = append(result, "")
		} else {
			result = append(result, prefix+line)
		}
	}
	return strings.Join(result, "\n")
}

// removeBlockFromSource removes the block text from src (including its trailing
// newline) and cleans up any resulting blank lines.
func removeBlockFromSource(src, block string) string {
	// Try to remove the block with a trailing newline
	result := strings.Replace(src, block+"\n", "", 1)
	if result == src {
		result = strings.Replace(src, block, "", 1)
	}
	// Collapse triple blank lines
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}
	// Remove blank line before closing brace
	result = regexp.MustCompile(`\n\n(\s*\})`).ReplaceAllString(result, "\n$1")
	return result
}

func awsMigrations() []Migration {
	return []Migration{
		{
			Namespace:   "hashicorp",
			Provider:    "aws",
			Name:        "v3-to-v4",
			Description: "Migrates AWS provider configuration from v3 to v4, refactoring S3 bucket sub-resources.",
			SubMigrations: []SubMigration{
				{
					Name:        "s3-bucket-acl",
					Description: "Extracts acl argument from aws_s3_bucket into a separate aws_s3_bucket_acl resource.",
					Apply:       applyS3BucketACL,
				},
				{
					Name:        "s3-bucket-cors",
					Description: "Extracts cors_rule block from aws_s3_bucket into a separate aws_s3_bucket_cors_configuration resource.",
					Apply:       applyS3BucketCORS,
				},
				{
					Name:        "s3-bucket-logging",
					Description: "Extracts logging block from aws_s3_bucket into a separate aws_s3_bucket_logging resource.",
					Apply:       applyS3BucketLogging,
				},
			},
		},
	}
}

var s3BucketResourceRe = regexp.MustCompile(`resource\s+"aws_s3_bucket"\s+"([^"]+)"`)
var aclLineRe = regexp.MustCompile(`(?m)^[ \t]*acl\s*=\s*"([^"]+)"[ \t]*\n`)

func applyS3BucketACL(_ string, src []byte) ([]byte, error) {
	s := string(src)

	m := s3BucketResourceRe.FindStringSubmatch(s)
	if m == nil {
		return src, nil
	}
	name := m[1]

	aclMatch := aclLineRe.FindStringSubmatch(s)
	if aclMatch == nil {
		return src, nil
	}
	aclValue := aclMatch[1]

	result := aclLineRe.ReplaceAllString(s, "")
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}

	newResource := fmt.Sprintf("\nresource \"aws_s3_bucket_acl\" %q {\n  bucket = aws_s3_bucket.%s.id\n  acl    = %q\n}\n", name, name, aclValue)
	result = result + newResource

	return []byte(result), nil
}

func applyS3BucketCORS(_ string, src []byte) ([]byte, error) {
	s := string(src)

	m := s3BucketResourceRe.FindStringSubmatch(s)
	if m == nil {
		return src, nil
	}
	name := m[1]

	fullBlock, _, _, _, ok := extractNestedBlock(s, "cors_rule")
	if !ok {
		return src, nil
	}

	result := removeBlockFromSource(s, fullBlock)

	// Dedent and re-indent the block at 2 spaces for the new resource
	dedented := dedentBlock(fullBlock)
	reindented := indentBlock(dedented, "  ")

	newResource := fmt.Sprintf("\nresource \"aws_s3_bucket_cors_configuration\" %q {\n  bucket = aws_s3_bucket.%s.id\n\n%s\n}\n", name, name, reindented)
	result = result + newResource

	return []byte(result), nil
}

func applyS3BucketLogging(_ string, src []byte) ([]byte, error) {
	s := string(src)

	m := s3BucketResourceRe.FindStringSubmatch(s)
	if m == nil {
		return src, nil
	}
	name := m[1]

	fullBlock, _, _, _, ok := extractNestedBlock(s, "logging")
	if !ok {
		return src, nil
	}

	result := removeBlockFromSource(s, fullBlock)

	dedented := dedentBlock(fullBlock)
	reindented := indentBlock(dedented, "  ")

	newResource := fmt.Sprintf("\nresource \"aws_s3_bucket_logging\" %q {\n  bucket = aws_s3_bucket.%s.id\n\n%s\n}\n", name, name, reindented)
	result = result + newResource

	return []byte(result), nil
}
