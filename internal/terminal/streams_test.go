// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terminal

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestStreamsFmtHelpers(t *testing.T) {
	streams, close := StreamsForTesting(t)

	streams.Print("stdout print ", 5, "\n")
	streams.Eprint("stderr print ", 6, "\n")
	streams.Println("stdout println", 7)
	streams.Eprintln("stderr println", 8)
	streams.Printf("stdout printf %d\n", 9)
	streams.Eprintf("stderr printf %d\n", 10)

	outp := close(t)

	gotOut := outp.Stdout()
	wantOut := `stdout print 5
stdout println 7
stdout printf 9
`
	if diff := cmp.Diff(wantOut, gotOut); diff != "" {
		t.Errorf("wrong stdout\n%s", diff)
	}

	gotErr := outp.Stderr()
	wantErr := `stderr print 6
stderr println 8
stderr printf 10
`
	if diff := cmp.Diff(wantErr, gotErr); diff != "" {
		t.Errorf("wrong stderr\n%s", diff)
	}
}
