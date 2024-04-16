// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package marks

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

func TestPathsWithMark(t *testing.T) {
	input := []cty.PathValueMarks{
		{
			Path:  cty.GetAttrPath("sensitive"),
			Marks: cty.NewValueMarks(Sensitive),
		},
		{
			Path:  cty.GetAttrPath("other"),
			Marks: cty.NewValueMarks("other"),
		},
		{
			Path:  cty.GetAttrPath("both"),
			Marks: cty.NewValueMarks(Sensitive, "other"),
		},
	}

	gotPaths, gotOthers := PathsWithMark(input, Sensitive)
	wantPaths := []cty.Path{
		cty.GetAttrPath("sensitive"),
		cty.GetAttrPath("both"),
	}
	wantOthers := []cty.PathValueMarks{
		{
			Path:  cty.GetAttrPath("other"),
			Marks: cty.NewValueMarks("other"),
		},
		{
			Path:  cty.GetAttrPath("both"),
			Marks: cty.NewValueMarks(Sensitive, "other"),
			// Note that this intentionally preserves the fact that the
			// attribute was both sensitive _and_ had another mark, since
			// that gives the caller the most possible information to
			// potentially handle this combination in a special way in
			// an error message, or whatever. It also conveniently avoids
			// allocating a new mark set, which is nice.
		},
	}

	if diff := cmp.Diff(wantPaths, gotPaths, ctydebug.CmpOptions); diff != "" {
		t.Errorf("wrong matched paths\n%s", diff)
	}
	if diff := cmp.Diff(wantOthers, gotOthers, ctydebug.CmpOptions); diff != "" {
		t.Errorf("wrong set of entries with other marks\n%s", diff)
	}
}
