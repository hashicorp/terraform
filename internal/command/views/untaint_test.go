// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestUntaintHuman_success(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	baseView := NewView(streams)
	baseView.Configure(&arguments.View{NoColor: true})
	v := NewUntaint(baseView)

	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_instance",
		Name: "foo",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	v.Success(addr)

	got := done(t).Stdout()
	if want := "Resource instance test_instance.foo has been successfully untainted."; !strings.Contains(got, want) {
		t.Errorf("wrong output\ngot:  %q\nwant: %q", got, want)
	}
}

func TestUntaintHuman_allowMissingWarning(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	baseView := NewView(streams)
	baseView.Configure(&arguments.View{NoColor: true})
	v := NewUntaint(baseView)

	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_instance",
		Name: "bar",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	v.AllowMissingWarning(addr)

	got := done(t).Stdout()
	if !strings.Contains(got, "No such resource instance") {
		t.Errorf("expected warning summary in output, got: %q", got)
	}
	if !strings.Contains(got, "test_instance.bar") {
		t.Errorf("expected resource address in output, got: %q", got)
	}
}

func TestUntaintHuman_diagnostics(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	baseView := NewView(streams)
	baseView.Configure(&arguments.View{NoColor: true})
	v := NewUntaint(baseView)

	diags := tfdiags.Diagnostics{
		tfdiags.Sourceless(
			tfdiags.Error,
			"Test error",
			"This is a test error message.",
		),
	}
	v.Diagnostics(diags)

	got := done(t).Stderr()
	if !strings.Contains(got, "Test error") {
		t.Errorf("expected error in stderr, got: %q", got)
	}
}

func TestUntaintHuman_helpPrompt(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	baseView := NewView(streams)
	baseView.Configure(&arguments.View{NoColor: true})
	v := NewUntaint(baseView)

	v.HelpPrompt()

	got := done(t).Stderr()
	if !strings.Contains(got, "terraform untaint -help") {
		t.Errorf("expected help prompt for untaint, got: %q", got)
	}
}
