package localexec

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvisioner_impl(t *testing.T) {
	var _ terraform.ResourceProvisioner = Provisioner()
}

func TestProvisioner(t *testing.T) {
	if err := Provisioner().(*schema.Provisioner).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestResourceProvider_Apply(t *testing.T) {
	defer os.Remove("test_out")
	c := testConfig(t, map[string]interface{}{
		"command": "echo foo > test_out",
	})

	output := new(terraform.MockUIOutput)
	p := Provisioner()

	if err := p.Apply(output, nil, c); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check the file
	raw, err := ioutil.ReadFile("test_out")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	actual := strings.TrimSpace(string(raw))
	expected := "foo"
	if actual != expected {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceProvider_stop(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		// bash/zsh/ksh will exec a single command in the same process. This
		// makes certain there's a subprocess in the shell.
		"command": "sleep 30; sleep 30",
	})

	output := new(terraform.MockUIOutput)
	p := Provisioner()

	var err error
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		err = p.Apply(output, nil, c)
	}()

	select {
	case <-doneCh:
		t.Fatal("should not finish quickly")
	case <-time.After(50 * time.Millisecond):
	}

	// Stop it
	p.Stop()

	select {
	case <-doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("should finish")
	}
}

func TestResourceProvider_Validate_good(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"command": "echo foo",
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

func TestResourceProvider_Validate_missing(t *testing.T) {
	c := testConfig(t, map[string]interface{}{})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) == 0 {
		t.Fatalf("Should have errors")
	}
}

func testConfig(t *testing.T, c map[string]interface{}) *terraform.ResourceConfig {
	r, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	return terraform.NewResourceConfig(r)
}
