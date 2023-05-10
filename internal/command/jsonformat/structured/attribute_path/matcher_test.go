// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
