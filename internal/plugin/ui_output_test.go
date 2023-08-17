// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin

import (
	"testing"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/mnptu/internal/mnptu"
)

func TestUIOutput_impl(t *testing.T) {
	var _ mnptu.UIOutput = new(UIOutput)
}

func TestUIOutput_input(t *testing.T) {
	client, server := plugin.TestRPCConn(t)
	defer client.Close()

	o := new(mnptu.MockUIOutput)

	err := server.RegisterName("Plugin", &UIOutputServer{
		UIOutput: o,
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	output := &UIOutput{Client: client}
	output.Output("foo")
	if !o.OutputCalled {
		t.Fatal("output should be called")
	}
	if o.OutputMessage != "foo" {
		t.Fatalf("bad: %#v", o.OutputMessage)
	}
}
