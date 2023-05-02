// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"context"
	"testing"
)

func TestPrefixUIInput_impl(t *testing.T) {
	var _ UIInput = new(PrefixUIInput)
}

func TestPrefixUIInput(t *testing.T) {
	input := new(MockUIInput)
	prefix := &PrefixUIInput{
		IdPrefix: "foo",
		UIInput:  input,
	}

	_, err := prefix.Input(context.Background(), &InputOpts{Id: "bar"})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if input.InputOpts.Id != "foo.bar" {
		t.Fatalf("bad: %#v", input.InputOpts)
	}
}
