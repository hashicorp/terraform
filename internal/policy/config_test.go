// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestConfigComparisonPaths(t *testing.T) {
	key := func(left, right string) RelationshipKey {
		return RelationshipKey{KeyPath: cty.GetAttrPath(left), ValuePath: cty.GetAttrPath(right)}
	}
	config := &Config{
		Definitions: map[string]Relationship{
			"unused": {
				KeyType: "test_target", ValueType: "test_target",
				Matches: []Match{{key("unused_key", "unused_value")}},
			},
		},
		Relationships: map[string][]Relationship{
			"test_other": {{
				Name: "reused", KeyType: "test_target", ValueType: "test_other",
				Matches: []Match{{key("outer_key", "id")}},
				Relationships: []Relationship{{
					Name: "reused", KeyType: "test_other", ValueType: "test_target",
					Matches: []Match{{key("id", "nested_value")}},
				}},
			}},
			"another_root": {{
				Name: "reused", KeyType: "test_other", ValueType: "test_target",
				Matches: []Match{{key("id", "other_tree")}},
			}, {
				Name: "same_type", KeyType: "test_target", ValueType: "test_target",
				Matches: []Match{
					{key("same_left", "same_right"), key("and_left", "and_right")},
					{key("or_left", "or_right")},
				},
			}},
		},
	}
	want := make(map[string]bool)
	for _, name := range []string{
		"outer_key", "nested_value", "other_tree", "same_left", "same_right",
		"and_left", "and_right", "or_left", "or_right",
	} {
		want[name] = true
	}
	paths := config.ComparisonPaths("test_target")
	if len(paths) != len(want) {
		t.Fatalf("got %d paths, want %d", len(paths), len(want))
	}
	for _, path := range paths {
		if len(path) != 1 {
			t.Errorf("unexpected comparison path %#v", path)
			continue
		}
		step, ok := path[0].(cty.IndexStep)
		if !ok {
			t.Errorf("comparison path is not canonical: %#v", path)
			continue
		}
		name := step.Key.AsString()
		if !want[name] {
			t.Errorf("unexpected or repeated comparison path %s", name)
		}
		delete(want, name)
	}
	if len(want) != 0 {
		t.Errorf("missing comparison paths: %v", want)
	}
	if paths := config.ComparisonPaths("test_unrelated"); len(paths) != 0 {
		t.Errorf("unrelated type received comparison paths: %#v", paths)
	}
}

func TestConfigComparisonPathsEmpty(t *testing.T) {
	for _, config := range []*Config{nil, {}} {
		if paths := config.ComparisonPaths("test_target"); len(paths) != 0 {
			t.Errorf("empty config received comparison paths: %#v", paths)
		}
	}
}
