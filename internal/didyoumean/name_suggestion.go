// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package didyoumean

import (
	"github.com/agext/levenshtein"
)

// maxCost refers to the max cost tolerated by the Levenshtein distance before
// it stops calculating. 3 was chosen as an experimental figure 6 years ago and has
// evidently been satisfactory since.
const maxCost = 3

// NameSuggestion tries to find a name from the given slice of suggested names
// that is close to the given name and returns it if found. If no suggestion
// is close enough, returns the empty string.
//
// The suggestions are tried in order, so earlier suggestions take precedence
// if the given string is similar to two or more suggestions.
//
// This function is intended to be used with a relatively-small number of
// suggestions. It's not optimized for hundreds or thousands of them.
func NameSuggestion(given string, suggestions []string) string {
	for _, suggestion := range suggestions {
		dist := levenshtein.Distance(given, suggestion, (levenshtein.NewParams()).MaxCost(maxCost))
		if dist < maxCost {
			return suggestion
		}
	}
	return ""
}
