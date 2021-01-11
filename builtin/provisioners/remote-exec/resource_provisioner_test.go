package remoteexec

import (
	"bytes"
	"context"
	"io"
	"log"
	"testing"
	"time"

	"strings"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/provisioners"
	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"
)

func TestResourceProvider_Validate_good(t *testing.T) {
	c := cty.ObjectVal(map[string]cty.Value{
		"inline": cty.ListVal([]cty.Value{cty.StringVal("echo foo")}),
	})

	resp := New().ValidateProvisionerConfig(provisioners.ValidateProvisionerConfigRequest{
		Config: c,
	})
	if len(resp.Diagnostics) > 0 {
		t.Fatal(resp.Diagnostics.ErrWithWarnings())
	}
}

func TestResourceProvider_Validate_bad(t *testing.T) {
	c := cty.ObjectVal(map[string]cty.Value{
		"invalid": cty.StringVal("nope"),
	})

	resp := New().ValidateProvisionerConfig(provisioners.ValidateProvisionerConfigRequest{
		Config: c,
	})
	if !resp.Diagnostics.HasErrors() {
		t.Fatalf("Should have errors")
	}
}

var expectedScriptOut = `cd /tmp
wget http://foobar
exit 0
`

func TestResourceProvider_generateScript(t *testing.T) {
	inline := cty.ListVal([]cty.Value{
		cty.StringVal("cd /tmp"),
		cty.StringVal("wget http://foobar"),
		cty.StringVal("exit 0"),
	})

	out, err := generateScripts(inline)
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
	inline := cty.ListVal([]cty.Value{cty.StringVal("")})

	_, err := generateScripts(inline)
	if err == nil {
		t.Fatal("expected error, got none")
	}

	if !strings.Contains(err.Error(), "empty string") {
		t.Fatalf("expected empty string error, got: %s", err)
	}
}

func TestResourceProvider_CollectScripts_inline(t *testing.T) {
	conf := map[string]cty.Value{
		"inline": cty.ListVal([]cty.Value{
			cty.StringVal("cd /tmp"),
			cty.StringVal("wget http://foobar"),
			cty.StringVal("exit 0"),
		}),
	}

	scripts, err := collectScripts(cty.ObjectVal(conf))
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
	p := New()
	schema := p.GetSchema().Provisioner

	conf, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		"scripts": cty.ListVal([]cty.Value{
			cty.StringVal("testdata/script1.sh"),
		}),
	}))
	if err != nil {
		t.Fatal(err)
	}

	scripts, err := collectScripts(conf)
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
	p := New()
	schema := p.GetSchema().Provisioner

	conf, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		"scripts": cty.ListVal([]cty.Value{
			cty.StringVal("testdata/script1.sh"),
			cty.StringVal("testdata/script1.sh"),
			cty.StringVal("testdata/script1.sh"),
		}),
	}))
	if err != nil {
		log.Fatal(err)
	}

	scripts, err := collectScripts(conf)
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
	p := New()
	schema := p.GetSchema().Provisioner

	conf, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		"scripts": cty.ListVal([]cty.Value{cty.StringVal("")}),
	}))
	if err != nil {
		t.Fatal(err)
	}

	_, err = collectScripts(conf)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "empty string") {
		t.Fatalf("Expected empty string error, got: %s", err)
	}
}

func TestProvisionerTimeout(t *testing.T) {
	o := cli.NewMockUi()
	c := new(communicator.MockCommunicator)

	disconnected := make(chan struct{})
	c.DisconnectFunc = func() error {
		close(disconnected)
		return nil
	}

	completed := make(chan struct{})
	c.CommandFunc = func(cmd *remote.Cmd) error {
		defer close(completed)
		cmd.Init()
		time.Sleep(2 * time.Second)
		cmd.SetExitStatus(0, nil)
		return nil
	}
	c.ConnTimeout = time.Second
	c.UploadScripts = map[string]string{"hello": "echo hello"}
	c.RemoteScriptPath = "hello"

	conf := map[string]cty.Value{
		"inline": cty.ListVal([]cty.Value{cty.StringVal("echo hello")}),
	}

	scripts, err := collectScripts(cty.ObjectVal(conf))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	done := make(chan struct{})

	var runErr error
	go func() {
		defer close(done)
		runErr = runScripts(ctx, o, c, scripts)
	}()

	select {
	case <-disconnected:
		t.Fatal("communicator disconnected before command completed")
	case <-completed:
	}

	<-done
	if runErr != nil {
		t.Fatal(err)
	}
}

// Validate that Stop can Close can be called even when not provisioning.
func TestResourceProvisioner_StopClose(t *testing.T) {
	p := New()
	p.Stop()
	p.Close()
}
