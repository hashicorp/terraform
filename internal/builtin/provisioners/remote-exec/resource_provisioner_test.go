package remoteexec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"sort"
	"testing"
	"time"

	"strings"

	"github.com/hashicorp/terraform/internal/communicator"
	"github.com/hashicorp/terraform/internal/communicator/remote"
	"github.com/hashicorp/terraform/internal/communicator/shared"
	"github.com/hashicorp/terraform/internal/provisioners"
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
	p := New()
	schema := p.GetSchema()
	v, err := schema.Provisioner.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		"inline": cty.ListVal([]cty.Value{
			cty.StringVal("cd /tmp"),
			cty.StringVal("wget http://foobar"),
			cty.StringVal("exit 0"),
		}),
	}))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := generateScripts(v.GetAttr("inline"), cty.ListValEmpty(cty.String))
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

func TestResourceProvider_generateEnvVars(t *testing.T) {
	cases := []struct {
		ConnectionType cty.Value
		EnvVars        cty.Value
		Want           cty.Value
		WantCount      int
	}{
		{
			cty.StringVal("ssh"),
			cty.ObjectVal(map[string]cty.Value{
				"TEST_VAR": cty.StringVal("My test var"),
			}),
			cty.ListVal([]cty.Value{cty.StringVal(`export TEST_VAR="My test var"`)}),
			1,
		},
		{
			cty.StringVal("ssh"),
			cty.ObjectVal(map[string]cty.Value{
				"TEST_VAR": cty.StringVal(`My test var with "quotes"`),
			}),
			cty.ListVal([]cty.Value{cty.StringVal(`export TEST_VAR="My test var with \"quotes\""`)}),
			1,
		},
		{
			cty.StringVal("ssh"),
			cty.ObjectVal(map[string]cty.Value{
				"TEST_VAR0": cty.StringVal("My test var"),
				"TEST_VAR1": cty.StringVal("My test var"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal(`export TEST_VAR0="My test var"`),
				cty.StringVal(`export TEST_VAR1="My test var"`),
			}),
			2,
		},
		{
			cty.StringVal("ssh"),
			cty.ObjectVal(map[string]cty.Value{
				"TEST_VAR0": cty.StringVal("My test var"),
				"TEST_VAR1": cty.StringVal(""),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal(`export TEST_VAR0="My test var"`),
			}),
			1,
		},
		{
			cty.StringVal("winrm"),
			cty.ObjectVal(map[string]cty.Value{
				"TEST_VAR0": cty.StringVal("My test var"),
			}),
			cty.ListVal([]cty.Value{
				cty.StringVal(`@set "TEST_VAR0=My test var"`),
			}),
			1,
		},
	}

	p := New()
	schema := p.GetSchema()

	for tn, test := range cases {
		t.Run(fmt.Sprintf("%d", tn), func(t *testing.T) {
			connection, err := shared.ConnectionBlockSupersetSchema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
				"type": test.ConnectionType,
				"host": cty.StringVal("127.0.0.1"),
			}))

			if err != nil {
				t.Fatal(err)
			}

			v, err := schema.Provisioner.CoerceValue(cty.ObjectVal(map[string]cty.Value{
				"environment": test.EnvVars,
			}))

			if err != nil {
				t.Fatal(err)
			}

			got := generateEnvVars(connection, v)

			if got.LengthInt() != test.WantCount {
				t.Fatalf("expected %d out, got %d\n", test.WantCount, got.LengthInt())
			}

			gotSlice := got.AsValueSlice()

			sort.Slice(gotSlice, func(i, j int) bool {
				return gotSlice[i].AsString() < gotSlice[j].AsString()
			})

			if !test.Want.RawEquals(cty.ListVal(gotSlice)) {
				t.Fatalf("Got unexpected result: %v instead of %v", got, test.Want)
			}
		})
	}
}

func TestResourceProvider_ScriptsWithEnvVars(t *testing.T) {
	c := cty.ObjectVal(map[string]cty.Value{
		"script": cty.StringVal("/dev/null"),
		"environment": cty.ObjectVal(map[string]cty.Value{
			"TEST_VAR": cty.StringVal("test"),
		}),
	})

	resp := New().ValidateProvisionerConfig(provisioners.ValidateProvisionerConfigRequest{
		Config: c,
	})

	if !resp.Diagnostics.HasErrors() {
		t.Fatalf("Should have errors")
	}
}

func TestResourceProvider_generateScriptEmptyInline(t *testing.T) {
	inline := cty.ListVal([]cty.Value{cty.StringVal("")})

	_, err := generateScripts(inline, cty.ListValEmpty(cty.String))
	if err == nil {
		t.Fatal("expected error, got none")
	}

	if !strings.Contains(err.Error(), "empty string") {
		t.Fatalf("expected empty string error, got: %s", err)
	}
}

func TestResourceProvider_CollectScripts_inline(t *testing.T) {
	p := New()
	schema := p.GetSchema().Provisioner

	conf, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		"inline": cty.ListVal([]cty.Value{
			cty.StringVal("cd /tmp"),
			cty.StringVal("wget http://foobar"),
			cty.StringVal("exit 0"),
		}),
	}))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	scripts, err := collectScripts(conf, cty.ListValEmpty(cty.String))
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

	scripts, err := collectScripts(conf, cty.ListValEmpty(cty.String))
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

	scripts, err := collectScripts(conf, cty.ListValEmpty(cty.String))
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

	_, err = collectScripts(conf, cty.ListValEmpty(cty.String))
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "empty string") {
		t.Fatalf("Expected empty string error, got: %s", err)
	}
}

func TestProvisionerTimeout(t *testing.T) {
	p := New()
	schema := p.GetSchema().Provisioner

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

	conf, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		"inline": cty.ListVal([]cty.Value{cty.StringVal("echo hello")}),
	}))
	if err != nil {
		t.Fatal(err)
	}

	scripts, err := collectScripts(conf, cty.ListValEmpty(cty.String))
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

func TestResourceProvisioner_connectionRequired(t *testing.T) {
	p := New()
	resp := p.ProvisionResource(provisioners.ProvisionResourceRequest{})
	if !resp.Diagnostics.HasErrors() {
		t.Fatal("expected error")
	}

	got := resp.Diagnostics.Err().Error()
	if !strings.Contains(got, "Missing connection") {
		t.Fatalf("expected 'Missing connection' error: got %q", got)
	}
}

func TestResourceProvisioner_nullsInOptionals(t *testing.T) {
	output := cli.NewMockUi()
	p := New()
	schema := p.GetSchema().Provisioner

	for i, cfg := range []cty.Value{
		cty.ObjectVal(map[string]cty.Value{
			"script": cty.StringVal("echo"),
			"inline": cty.NullVal(cty.List(cty.String)),
		}),
		cty.ObjectVal(map[string]cty.Value{
			"inline": cty.ListVal([]cty.Value{
				cty.NullVal(cty.String),
			}),
		}),
		cty.ObjectVal(map[string]cty.Value{
			"script": cty.NullVal(cty.String),
		}),
		cty.ObjectVal(map[string]cty.Value{
			"scripts": cty.NullVal(cty.List(cty.String)),
		}),
		cty.ObjectVal(map[string]cty.Value{
			"scripts": cty.ListVal([]cty.Value{
				cty.NullVal(cty.String),
			}),
		}),
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {

			cfg, err := schema.CoerceValue(cfg)
			if err != nil {
				t.Fatal(err)
			}

			// verifying there are no panics
			p.ProvisionResource(provisioners.ProvisionResourceRequest{
				Config:   cfg,
				UIOutput: output,
			})
		})
	}
}
