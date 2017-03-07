package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
)

func TestSuppressEquivalentJsonDiffsWhitespaceAndNoWhitespace(t *testing.T) {
	d := new(schema.ResourceData)

	noWhitespace := `{"test":"test"}`
	whitespace := `
{
  "test": "test"
}`

	if !suppressEquivalentJsonDiffs("", noWhitespace, whitespace, d) {
		t.Errorf("Expected suppressEquivalentJsonDiffs to return true for %s == %s", noWhitespace, whitespace)
	}

	noWhitespaceDiff := `{"test":"test"}`
	whitespaceDiff := `
{
  "test": "tested"
}`

	if suppressEquivalentJsonDiffs("", noWhitespaceDiff, whitespaceDiff, d) {
		t.Errorf("Expected suppressEquivalentJsonDiffs to return false for %s == %s", noWhitespaceDiff, whitespaceDiff)
	}
}
