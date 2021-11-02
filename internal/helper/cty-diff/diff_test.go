package cty_diff

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestListDiff(t *testing.T) {
	testCases := []distanceTestCase{
		{"ABC", "ABC", "A=A B=B C=C", func(distance float32) bool { return distance == 0 }},
		{"ABC", "ADC", "A=A B~D C=C", func(distance float32) bool { return distance == float32(1)/float32(3) }},
		{"ABC", "AC", "A=A B-_ C=C", func(distance float32) bool { return distance == float32(1)/float32(3) }},
		{"AC", "ABC", "A=A _+B C=C", func(distance float32) bool { return distance == float32(1)/float32(3) }},
		{"", "A", "_+A", func(distance float32) bool { return distance == 1 }},
		{"A", "", "A-_", func(distance float32) bool { return distance == 1 }},
	}
	for _, tc := range testCases {
		var lists [2]cty.Value
		for i, s := range [2]string{tc.a, tc.b} {
			if s == "" {
				lists[i] = cty.ListValEmpty(cty.String)
			} else {
				list := make([]cty.Value, len(s))
				for j := range s {
					list[j] = cty.StringVal(s[j : j+1])
				}
				lists[i] = cty.ListVal(list)
			}
		}

		distance, path := ListDiff(lists[0], lists[1], true)
		checkResult(t, &tc, distance, path)
	}
}

func TestSetDiff(t *testing.T) {
	testCases := []distanceTestCase{
		{"ABC", "ABC", "A=A B=B C=C", func(distance float32) bool { return distance == 0 }},
		{"ABC", "ADC", "A=A C=C B-_ _+D", func(distance float32) bool { return distance == float32(1)/float32(2) }},
		{"ABC", "AC", "A=A C=C B-_", func(distance float32) bool { return distance == float32(1)/float32(3) }},
		{"AC", "ABC", "A=A C=C _+B", func(distance float32) bool { return distance == float32(1)/float32(3) }},
		{"", "A", "_+A", func(distance float32) bool { return distance == 1 }},
		{"A", "", "A-_", func(distance float32) bool { return distance == 1 }},
	}
	for _, tc := range testCases {
		var lists [2]cty.Value
		for i, s := range [2]string{tc.a, tc.b} {
			if s == "" {
				lists[i] = cty.SetValEmpty(cty.String)
			} else {
				list := make([]cty.Value, len(s))
				for j := range s {
					list[j] = cty.StringVal(s[j : j+1])
				}
				lists[i] = cty.SetVal(list)
			}
		}

		distance, path := SetDiff(lists[0], lists[1], true)
		checkResult(t, &tc, distance, path)
	}
}

func TestMapDiff(t *testing.T) {
	testCases := []distanceTestCase{
		{"ABC", "ABC", "A=A B=B C=C", func(distance float32) bool { return distance == 0 }},
		{"ABC", "ADC", "A=A B~D C=C", func(distance float32) bool { return distance == float32(1)/float32(3) }},
		{"ABC", "AB", "A=A B=B C-_", func(distance float32) bool { return distance == float32(1)/float32(3) }},
		{"AB", "ABC", "A=A B=B _+C", func(distance float32) bool { return distance == float32(1)/float32(3) }},
		{"", "A", "_+A", func(distance float32) bool { return distance == 1 }},
		{"A", "", "A-_", func(distance float32) bool { return distance == 1 }},
	}
	for _, tc := range testCases {
		var lists [2]cty.Value
		for i, s := range [2]string{tc.a, tc.b} {
			if s == "" {
				lists[i] = cty.MapValEmpty(cty.String)
			} else {
				list := make(map[string]cty.Value, len(s))
				for j := range s {
					list[fmt.Sprintf("%d", j)] = cty.StringVal(s[j : j+1])
				}
				lists[i] = cty.MapVal(list)
			}
		}

		distance, path := MapDiff(lists[0], lists[1], true)
		checkResult(t, &tc, distance, path)
	}
}

type distanceTestCase struct {
	a         string
	b         string
	path      string
	checkDiff func(distance float32) bool
}

func checkResult(t *testing.T, tc *distanceTestCase, distance float32, path []EditStep) {
	if !tc.checkDiff(distance) {
		t.Errorf("%q -> %q: Unexpected distance %f", tc.a, tc.b, distance)
	}

	pathStrs := make([]string, len(path))
	for i, step := range path {
		old := "_"
		if step.OldValue != (cty.Value{}) {
			old = step.OldValue.AsString()
		}
		new := "_"
		if step.NewValue != (cty.Value{}) {
			new = step.NewValue.AsString()
		}
		pathStrs[i] = fmt.Sprintf("%s%c%s", old, step.Operation, new)
	}
	pathStr := strings.Join(pathStrs, " ")
	if pathStr != tc.path {
		t.Errorf("%q -> %q: Unexpected path %s; wanted %s", tc.a, tc.b, pathStr, tc.path)
	}
}
