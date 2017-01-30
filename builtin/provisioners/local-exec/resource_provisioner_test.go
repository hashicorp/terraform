package localexec

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceProvisioner_impl(t *testing.T) {
	var _ terraform.ResourceProvisioner = new(ResourceProvisioner)
}

func TestResourceProvider_Apply(t *testing.T) {
	defer os.Remove("test_out")
	c := testConfig(t, map[string]interface{}{
		"command": "echo foo > test_out",
	})

	output := new(terraform.MockUIOutput)
	p := new(ResourceProvisioner)
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

func TestResourceProvider_Validate_good(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"command": "echo foo",
	})
	p := new(ResourceProvisioner)
	warn, errs := p.Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

func TestResourceProvider_Validate_missing(t *testing.T) {
	c := testConfig(t, map[string]interface{}{})
	p := new(ResourceProvisioner)
	warn, errs := p.Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) == 0 {
		t.Fatalf("Should have errors")
	}
}

func TestResourceProvider_Verify(t *testing.T) {
	// Setup the file, containing 'foo'
	defer os.Remove("test_out")
	c := testConfig(t, map[string]interface{}{
		"command": "echo bar > test_out",
		"verify":  "grep -q bar test_out",
	})

	output := new(terraform.MockUIOutput)
	p := new(ResourceProvisioner)
	if err := p.Apply(output, nil, c); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check the file
	raw, err := ioutil.ReadFile("test_out")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	actual := strings.TrimSpace(string(raw))
	expected := "bar"
	if actual != expected {
		t.Fatalf("bad: %#v", actual)
	}
}

// Not actually a failure, just forces execution
func TestResourceProvider_VerifyFail(t *testing.T) {
	// Setup the file, containing 'bar'
	defer os.Remove("test_out")
	c := testConfig(t, map[string]interface{}{
		"command": "echo bar > test_out",
		"verify":  "grep -q foo test_out",
	})

	output := new(terraform.MockUIOutput)
	p := new(ResourceProvisioner)
	if err := p.Apply(output, nil, c); err == nil {
		t.Fatalf("should have failed")
		if !strings.Contains(err.Error(), "verifying") {
			t.Fatalf("should have failed at verifying: %v", err)
		}
	}

	// Check the file
	raw, err := ioutil.ReadFile("test_out")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	actual := strings.TrimSpace(string(raw))
	expected := "bar"
	if actual != expected {
		t.Fatalf("bad: %#v", actual)
	}
}

func testConfig(
	t *testing.T,
	c map[string]interface{}) *terraform.ResourceConfig {
	r, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	return terraform.NewResourceConfig(r)
}
