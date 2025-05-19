// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package logging

import (
	"testing"
)

func TestIndent(t *testing.T) {
	s := "hello\n  world\ngoodbye\n  moon"
	got := Indent(s)
	want := "  hello\n    world\n  goodbye\n    moon"

	if got != want {
		t.Errorf("wrong result\ngot:\n%s\n\nwant:\n%s", got, want)
	}
}
