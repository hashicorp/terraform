// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"testing"
)

func TestMostSpecific(t *testing.T) {
	contains := func(a, b TargetPattern) bool {
		return a.Contains(b)
	}

	for name, tc := range map[string]struct {
		candidates []string
		want       string
	}{
		"none": {},
		"one": {
			candidates: []string{"module.a"},
			want:       "module.a",
		},
		"nested": {
			candidates: []string{"module.a", "module.a[1].test.b", "module.a[1]", "module.a.test.b"},
			want:       "module.a[1].test.b",
		},
		"nested in reverse": {
			candidates: []string{"module.a[1].test.b", "module.a[1]", "module.a"},
			want:       "module.a[1].test.b",
		},
		"equivalent": {
			// both select every instance, so the first takes precedence
			candidates: []string{"module.a", "module.a[*]"},
			want:       "module.a",
		},
		"not nested": {
			// neither contains the other, so the first takes precedence
			candidates: []string{"module.a[0].test.b", "module.a[1].test.b"},
			want:       "module.a[0].test.b",
		},
	} {
		t.Run(name, func(t *testing.T) {
			var candidates []TargetPattern
			for _, s := range tc.candidates {
				candidates = append(candidates, mustParseTargetPattern(s).(TargetPattern))
			}

			got, ok := MostSpecific(candidates, contains)
			if ok != (tc.want != "") {
				t.Fatalf("wrong ok: got %t", ok)
			}
			if ok && got.String() != tc.want {
				t.Errorf("wrong result\ngot:  %s\nwant: %s", got, tc.want)
			}
		})
	}
}
