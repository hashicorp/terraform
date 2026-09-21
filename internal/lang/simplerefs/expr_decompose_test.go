// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package simplerefs

import (
	"testing"
)

// TestDecomposeMembershipTraversals verifies the expansion-connector
// decomposition: coexisting collection constructs yield every element's
// reference, while *choice* constructs (conditionals, selection functions) are
// rejected (ok=false) so the caller defers/halts instead of over-claiming
// membership.
func TestDecomposeMembershipTraversals(t *testing.T) {
	cases := map[string]struct {
		src      string
		wantOK   bool
		wantRefs int // number of decomposed traversals when ok
	}{
		"tuple of refs":        {src: `[aws_subnet.a.id, aws_subnet.b.id]`, wantOK: true, wantRefs: 2},
		"splat":                {src: `aws_subnet.this[*].id`, wantOK: true, wantRefs: 1},
		"concat of splats":     {src: `concat(aws_subnet.a[*].id, aws_subnet.b[*].id)`, wantOK: true, wantRefs: 2},
		"flatten":              {src: `flatten([aws_subnet.a.id, aws_subnet.b.id])`, wantOK: true, wantRefs: 2},
		"toset":                {src: `toset([aws_subnet.a.id])`, wantOK: true, wantRefs: 1},
		"for over collection":  {src: `[for s in aws_subnet.this : s.id]`, wantOK: true, wantRefs: 1},
		"tuple with a literal": {src: `[aws_subnet.a.id, "subnet-hardcoded"]`, wantOK: true, wantRefs: 1},

		// Choice constructs are rejected: their members are not all present.
		"conditional of collections": {src: `true ? [aws_subnet.a.id] : [aws_subnet.b.id]`, wantOK: false},
		"element selects one":        {src: `[element(aws_subnet.this[*].id, 0)]`, wantOK: false},
		"try chooses":                {src: `[try(aws_subnet.a.id, aws_subnet.b.id)]`, wantOK: false},
		"coalesce chooses":           {src: `coalesce(aws_subnet.a[*].id, aws_subnet.b[*].id)`, wantOK: false},
		"slice drops by index":       {src: `slice(aws_subnet.this[*].id, 0, 1)`, wantOK: false},
		"opaque function":            {src: `[upper(aws_subnet.a.id)]`, wantOK: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			travs, _, ok := DecomposeMembershipTraversals(exprFor(t, tc.src))
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v (src %q)", ok, tc.wantOK, tc.src)
			}
			if tc.wantOK && len(travs) != tc.wantRefs {
				t.Fatalf("decomposed %d traversals, want %d (src %q): %v", len(travs), tc.wantRefs, tc.src, travs)
			}
		})
	}
}
