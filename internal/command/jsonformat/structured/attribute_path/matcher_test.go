// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package attribute_path

import "testing"

func TestPathMatcher_FollowsPath(t *testing.T) {
	var matcher Matcher

	matcher = &PathMatcher{
		Paths: [][]interface{}{
			{
				float64(0),
				"key",
				float64(0),
			},
		},
	}

	if matcher.Matches() {
		t.Errorf("should not have exact matched at base level")
	}
	if !matcher.MatchesPartial() {
		t.Errorf("should have partial matched at base level")
	}

	matcher = matcher.GetChildWithIndex(0)

	if matcher.Matches() {
		t.Errorf("should not have exact matched at first level")
	}
	if !matcher.MatchesPartial() {
		t.Errorf("should have partial matched at first level")
	}

	matcher = matcher.GetChildWithKey("key")

	if matcher.Matches() {
		t.Errorf("should not have exact matched at second level")
	}
	if !matcher.MatchesPartial() {
		t.Errorf("should have partial matched at second level")
	}

	matcher = matcher.GetChildWithIndex(0)

	if !matcher.Matches() {
		t.Errorf("should have exact matched at leaf level")
	}
	if !matcher.MatchesPartial() {
		t.Errorf("should have partial matched at leaf level")
	}
}
func TestPathMatcher_Propagates(t *testing.T) {
	var matcher Matcher

	matcher = &PathMatcher{
		Paths: [][]interface{}{
			{
				float64(0),
				"key",
			},
		},
		Propagate: true,
	}

	if matcher.Matches() {
		t.Errorf("should not have exact matched at base level")
	}
	if !matcher.MatchesPartial() {
		t.Errorf("should have partial matched at base level")
	}

	matcher = matcher.GetChildWithIndex(0)

	if matcher.Matches() {
		t.Errorf("should not have exact matched at first level")
	}
	if !matcher.MatchesPartial() {
		t.Errorf("should have partial matched at first level")
	}

	matcher = matcher.GetChildWithKey("key")

	if !matcher.Matches() {
		t.Errorf("should have exact matched at second level")
	}
	if !matcher.MatchesPartial() {
		t.Errorf("should have partial matched at second level")
	}

	matcher = matcher.GetChildWithIndex(0)

	if !matcher.Matches() {
		t.Errorf("should have exact matched at leaf level")
	}
	if !matcher.MatchesPartial() {
		t.Errorf("should have partial matched at leaf level")
	}
}
func TestPathMatcher_DoesNotPropagate(t *testing.T) {
	var matcher Matcher

	matcher = &PathMatcher{
		Paths: [][]interface{}{
			{
				float64(0),
				"key",
			},
		},
	}

	if matcher.Matches() {
		t.Errorf("should not have exact matched at base level")
	}
	if !matcher.MatchesPartial() {
		t.Errorf("should have partial matched at base level")
	}

	matcher = matcher.GetChildWithIndex(0)

	if matcher.Matches() {
		t.Errorf("should not have exact matched at first level")
	}
	if !matcher.MatchesPartial() {
		t.Errorf("should have partial matched at first level")
	}

	matcher = matcher.GetChildWithKey("key")

	if !matcher.Matches() {
		t.Errorf("should have exact matched at second level")
	}
	if !matcher.MatchesPartial() {
		t.Errorf("should have partial matched at second level")
	}

	matcher = matcher.GetChildWithIndex(0)

	if matcher.Matches() {
		t.Errorf("should not have exact matched at leaf level")
	}
	if matcher.MatchesPartial() {
		t.Errorf("should not have partial matched at leaf level")
	}
}

func TestPathMatcher_BreaksPath(t *testing.T) {
	var matcher Matcher

	matcher = &PathMatcher{
		Paths: [][]interface{}{
			{
				float64(0),
				"key",
				float64(0),
			},
		},
	}

	if matcher.Matches() {
		t.Errorf("should not have exact matched at base level")
	}
	if !matcher.MatchesPartial() {
		t.Errorf("should have partial matched at base level")
	}

	matcher = matcher.GetChildWithIndex(0)

	if matcher.Matches() {
		t.Errorf("should not have exact matched at first level")
	}
	if !matcher.MatchesPartial() {
		t.Errorf("should have partial matched at first level")
	}

	matcher = matcher.GetChildWithKey("invalid")

	if matcher.Matches() {
		t.Errorf("should not have exact matched at second level")
	}
	if matcher.MatchesPartial() {
		t.Errorf("should not have partial matched at second level")

	}
}

func TestPathMatcher_MultiplePaths(t *testing.T) {
	var matcher Matcher

	matcher = &PathMatcher{
		Paths: [][]interface{}{
			{
				float64(0),
				"key",
				float64(0),
			},
			{
				float64(0),
				"key",
				float64(1),
			},
		},
	}

	if matcher.Matches() {
		t.Errorf("should not have exact matched at base level")
	}
	if !matcher.MatchesPartial() {
		t.Errorf("should have partial matched at base level")
	}

	matcher = matcher.GetChildWithIndex(0)

	if matcher.Matches() {
		t.Errorf("should not have exact matched at first level")
	}
	if !matcher.MatchesPartial() {
		t.Errorf("should have partial matched at first level")
	}

	matcher = matcher.GetChildWithKey("key")

	if matcher.Matches() {
		t.Errorf("should not have exact matched at second level")
	}
	if !matcher.MatchesPartial() {
		t.Errorf("should have partial matched at second level")
	}

	validZero := matcher.GetChildWithIndex(0)
	validOne := matcher.GetChildWithIndex(1)
	invalid := matcher.GetChildWithIndex(2)

	if !validZero.Matches() {
		t.Errorf("should have exact matched at leaf level")
	}
	if !validZero.MatchesPartial() {
		t.Errorf("should have partial matched at leaf level")
	}

	if !validOne.Matches() {
		t.Errorf("should have exact matched at leaf level")
	}
	if !validOne.MatchesPartial() {
		t.Errorf("should have partial matched at leaf level")
	}

	if invalid.Matches() {
		t.Errorf("should not have exact matched at leaf level")
	}
	if invalid.MatchesPartial() {
		t.Errorf("should not have partial matched at leaf level")
	}
}

// Since paths may be coming from relevant attributes, and those paths may no
// longer correspond to an updated schema, we can't always be certain the caller
// knows the correct type.
func TestPathMatcher_WrongKeyTypes(t *testing.T) {
	var matcher Matcher

	matcher = &PathMatcher{
		Paths: [][]interface{}{
			{
				float64(0),
				"key",
				float64(0),
			},
		},
	}

	failed := matcher.GetChildWithKey("key")
	if failed.Matches() || failed.MatchesPartial() {
		t.Errorf("should not have any match at on failure")
	}

	matcher = matcher.GetChildWithIndex(0).GetChildWithKey("key")

	if matcher.Matches() {
		t.Errorf("should not have exact matched at first level")
	}
	if !matcher.MatchesPartial() {
		t.Errorf("should have partial matched at first level")
	}

	failed = matcher.GetChildWithKey("zero")
	if failed.Matches() || failed.MatchesPartial() {
		t.Errorf("should not have any match at on failure")
	}
}
