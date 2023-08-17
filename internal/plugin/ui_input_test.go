// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/mnptu/internal/mnptu"
)

func TestUIInput_impl(t *testing.T) {
	var _ mnptu.UIInput = new(UIInput)
}

func TestUIInput_input(t *testing.T) {
	client, server := plugin.TestRPCConn(t)
	defer client.Close()

	i := new(mnptu.MockUIInput)
	i.InputReturnString = "foo"

	err := server.RegisterName("Plugin", &UIInputServer{
		UIInput: i,
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	input := &UIInput{Client: client}

	opts := &mnptu.InputOpts{
		Id: "foo",
	}

	v, err := input.Input(context.Background(), opts)
	if !i.InputCalled {
		t.Fatal("input should be called")
	}
	if !reflect.DeepEqual(i.InputOpts, opts) {
		t.Fatalf("bad: %#v", i.InputOpts)
	}
	if err != nil {
		t.Fatalf("bad: %#v", err)
	}

	if v != "foo" {
		t.Fatalf("bad: %#v", v)
	}
}
