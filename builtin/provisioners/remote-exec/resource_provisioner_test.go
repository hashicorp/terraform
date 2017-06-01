package remoteexec

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"strings"

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

func TestResourceProvider_Validate_good(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"inline": "echo foo",
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

func TestResourceProvider_Validate_bad(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"invalid": "nope",
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) == 0 {
		t.Fatalf("Should have errors")
	}
}

var expectedScriptOut = `cd /tmp
wget http://foobar
exit 0
`

func TestResourceProvider_generateScript(t *testing.T) {
	conf := map[string]interface{}{
		"inline": []interface{}{
			"cd /tmp",
			"wget http://foobar",
			"exit 0",
		},
	}

	out, err := generateScripts(
		schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, conf),
	)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if len(out) != 1 {
		t.Fatal("expected 1 out")
	}

	if out[0] != expectedScriptOut {
		t.Fatalf("bad: %v", out)
	}
}

func TestResourceProvider_generateScriptEmptyInline(t *testing.T) {
	p := Provisioner().(*schema.Provisioner)
	conf := map[string]interface{}{
		"inline": []interface{}{""},
	}

	_, err := generateScripts(schema.TestResourceDataRaw(
		t, p.Schema, conf))
	if err == nil {
		t.Fatal("expected error, got none")
	}

	if !strings.Contains(err.Error(), "Error parsing") {
		t.Fatalf("expected parsing error, got: %s", err)
	}
}

func TestResourceProvider_CollectScripts_inline(t *testing.T) {
	conf := map[string]interface{}{
		"inline": []interface{}{
			"cd /tmp",
			"wget http://foobar",
			"exit 0",
		},
	}

	scripts, err := collectScripts(
		schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, conf),
	)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if len(scripts) != 1 {
		t.Fatalf("bad: %v", scripts)
	}

	var out bytes.Buffer
	_, err = io.Copy(&out, scripts[0])
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if out.String() != expectedScriptOut {
		t.Fatalf("bad: %v", out.String())
	}
}

func TestResourceProvider_CollectScripts_script(t *testing.T) {
	conf := map[string]interface{}{
		"script": "test-fixtures/script1.sh",
	}

	scripts, err := collectScripts(
		schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, conf),
	)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if len(scripts) != 1 {
		t.Fatalf("bad: %v", scripts)
	}

	var out bytes.Buffer
	_, err = io.Copy(&out, scripts[0])
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if out.String() != expectedScriptOut {
		t.Fatalf("bad: %v", out.String())
	}
}

func TestResourceProvider_CollectScripts_scripts(t *testing.T) {
	conf := map[string]interface{}{
		"scripts": []interface{}{
			"test-fixtures/script1.sh",
			"test-fixtures/script1.sh",
			"test-fixtures/script1.sh",
		},
	}

	scripts, err := collectScripts(
		schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, conf),
	)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if len(scripts) != 3 {
		t.Fatalf("bad: %v", scripts)
	}

	for idx := range scripts {
		var out bytes.Buffer
		_, err = io.Copy(&out, scripts[idx])
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		if out.String() != expectedScriptOut {
			t.Fatalf("bad: %v", out.String())
		}
	}
}

func TestResourceProvider_CollectScripts_scriptsEmpty(t *testing.T) {
	p := Provisioner().(*schema.Provisioner)
	conf := map[string]interface{}{
		"scripts": []interface{}{""},
	}

	_, err := collectScripts(schema.TestResourceDataRaw(
		t, p.Schema, conf))

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "Error parsing") {
		t.Fatalf("Expected parsing error, got: %s", err)
	}
}

func TestRetryFunc(t *testing.T) {
	// succeed on the third try
	errs := []error{io.EOF, &net.OpError{Err: errors.New("ERROR")}, nil}
	count := 0

	err := retryFunc(context.Background(), time.Second, func() error {
		if count >= len(errs) {
			return errors.New("failed to stop after nil error")
		}

		err := errs[count]
		count++

		return err
	})

	if count != 3 {
		t.Fatal("retry func should have been called 3 times")
	}

	if err != nil {
		t.Fatal(err)
	}
}

func testConfig(t *testing.T, c map[string]interface{}) *terraform.ResourceConfig {
	r, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	return terraform.NewResourceConfig(r)
}
