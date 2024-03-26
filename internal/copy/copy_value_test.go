// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package copy

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
)

func TestCopyValue(t *testing.T) {
	t.Run("pointer to something that needs copying", func(t *testing.T) {
		// To test this we need to point to something that actually gets
		// deep copied, because the pointer _itself_ is just a number,
		// not mutably-aliased memory. (If the pointee is not something
		// that can be mutably aliased then the result would just match the
		// input, because no copying is needed.)
		type V struct {
			S string
		}
		input := &V{"hello"}
		result := testDeepCopyValueLogged(t, input)
		if input == result {
			t.Errorf("result pointer matches input pointer")
		}
		if input.S != "hello" {
			t.Errorf("input was modified before we modified it")
		}
		result.S = "goodbye"
		if input.S != "hello" {
			t.Errorf("modifying result also modified input")
		}
	})
	t.Run("pointer to something that doesn't need copying", func(t *testing.T) {
		// Strings are immutable and so we don't deep-copy them. Therefore
		// a pointer to a string doesn't get modified during copy either.
		s := "hello"
		input := &s
		result := testDeepCopyValueLogged(t, input)
		if input != result {
			t.Errorf("result pointer does not match input pointer")
		}
	})
	t.Run("pointer that is nil", func(t *testing.T) {
		var input *int
		result := testDeepCopyValueLogged(t, input)
		if result != nil {
			t.Errorf("result is not nil")
		}
	})
	t.Run("slice", func(t *testing.T) {
		arr := [...]rune{'a', 'b', 'c', 'd'}
		input := arr[0:2:4] // ab is in length, cd is hidden in extra capacity
		result := testDeepCopyValueLogged(t, input)
		if &input[0] == &result[0] {
			t.Errorf("result shares backing array with input")
		}
		if got := len(result); got != 2 {
			t.Fatalf("result has incorrect length %d", got)
		}
		if got := cap(result); got != 4 {
			t.Fatalf("result has incorrect capacity %d", got)
		}
		// We'll expand the slices so we can view the excess capacity too
		fullInput := input[0:4]
		fullResult := result[0:4]
		want := []rune{'a', 'b', 'c', 'd'}
		if diff := cmp.Diff(want, fullInput); diff != "" {
			t.Errorf("input was modified\n%s", diff)
		}
		if diff := cmp.Diff(want, fullResult); diff != "" {
			t.Errorf("incorrect result\n%s", diff)
		}
	})
	t.Run("slice that is nil", func(t *testing.T) {
		var input []int
		result := testDeepCopyValueLogged(t, input)
		if result != nil {
			t.Errorf("result is not nil")
		}
	})
	t.Run("array", func(t *testing.T) {
		// Arrays are passed by value anyway, so deep copying one really
		// means deep copying anything they refer to that might contain
		// mutably-aliased data. We'll use slices as the victims here;
		// their backing arrays should be copied and thus the result
		// should have different slices but with the same content.
		input := [...][]rune{
			{'a', 'b'},
			{'c', 'd'},
		}
		result := testDeepCopyValueLogged(t, input)
		if &result[0][0] == &input[0][0] {
			t.Errorf("first element of result shares backing array with input")
		}
		if &result[1][0] == &input[1][0] {
			t.Errorf("second element of result shares backing array with input")
		}
		want := [...][]rune{
			{'a', 'b'},
			{'c', 'd'},
		}
		if diff := cmp.Diff(want, result); diff != "" {
			t.Errorf("incorrect result\n%s", diff)
		}
	})
	t.Run("map", func(t *testing.T) {
		// Maps are a bit tricky to test because they are an address-based
		// data structure but the addresses of the internals are intentionally
		// not exposed. Therefore we'll test this indirectly by making a
		// map, copying it, and then modifying the copy. That should leave
		// the original unchanged, if the copy was performed correctly.
		input := map[string]string{"greeting": "hello"}
		result := testDeepCopyValueLogged(t, input)
		if len(input) != 1 {
			t.Errorf("input length changed before we did any modifying")
		}
		if input["greeting"] != "hello" {
			t.Errorf("input element changed before we did any modifying")
		}
		if len(result) != 1 {
			t.Errorf("result length changed before we did any modifying")
		}
		if result["greeting"] != "hello" {
			t.Errorf("result element changed before we did any modifying")
		}
		result["greeting"] = "hallo"
		if input["greeting"] != "hello" {
			t.Errorf("input element changed when we modified result")
		}
	})
	t.Run("map that is nil", func(t *testing.T) {
		var input map[string]string
		result := testDeepCopyValueLogged(t, input)
		if result != nil {
			t.Errorf("result is not nil")
		}
	})
	t.Run("struct", func(t *testing.T) {
		type S struct {
			Exported   string
			unexported string
		}
		input := S{
			Exported:   "beep",
			unexported: "boop",
		}
		result := testDeepCopyValueLogged(t, input)
		if result.Exported != "beep" {
			t.Errorf("Exported field has wrong result")
		}
		if result.unexported != "" {
			t.Errorf("unexported field got populated (should have been left as zero value)")
		}
	})
	t.Run("interface", func(t *testing.T) {
		// We'll create an interface that contains a pointer to something
		// mutable, and then mutate it after copy to make sure that the
		// two values can change independently.
		type B struct {
			S string
		}
		type A struct {
			B *B
		}
		inputInner := &A{
			&B{"hello"},
		}
		input := any(inputInner) // an interface value wrapping inputInner
		result := testDeepCopyValueLogged(t, input)
		if resultInner, ok := result.(*A); !ok {
			t.Fatalf("result contains %T, not %T", result, resultInner)
		}
		if result.(*A) == input.(*A) {
			t.Error("result has same address as input")
		}
		if result.(*A).B == input.(*A).B {
			t.Error("result.b has same address as input")
		}
		if input.(*A).B.S != "hello" {
			t.Errorf("input was modified before we modified it")
		}
		result.(*A).B.S = "goodbye"
		if input.(*A).B.S != "hello" {
			t.Errorf("modifying result also modified input")
		}
	})
}

func testDeepCopyValueLogged[T any](t *testing.T, input T) T {
	t.Helper()
	t.Logf("input:  %s", spew.Sdump(input))
	result := DeepCopyValue(input)
	t.Logf("result: %s", spew.Sdump(result))
	return result
}
