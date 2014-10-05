package rpc

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestUIOutput_impl(t *testing.T) {
	var _ terraform.UIOutput = new(UIOutput)
}

func TestUIOutput_input(t *testing.T) {
	client, server := testClientServer(t)
	defer client.Close()

	o := new(terraform.MockUIOutput)

	err := server.RegisterName("UIOutput", &UIOutputServer{
		UIOutput: o,
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	output := &UIOutput{Client: client, Name: "UIOutput"}
	output.Output("foo")
	if !o.OutputCalled {
		t.Fatal("output should be called")
	}
	if o.OutputMessage != "foo" {
		t.Fatalf("bad: %#v", o.OutputMessage)
	}
}
