package localexec

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"
)

func TestResourceProvider_Apply(t *testing.T) {
	defer os.Remove("test_out")
	output := cli.NewMockUi()
	p := New()
	schema := p.GetSchema().Provisioner
	c, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		"command": cty.StringVal("echo foo > test_out"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	resp := p.ProvisionResource(provisioners.ProvisionResourceRequest{
		Config:   c,
		UIOutput: output,
	})

	if resp.Diagnostics.HasErrors() {
		t.Fatalf("err: %v", resp.Diagnostics.Err())
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
	output := cli.NewMockUi()
	p := New()
	schema := p.GetSchema().Provisioner

	c, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		// bash/zsh/ksh will exec a single command in the same process. This
		// makes certain there's a subprocess in the shell.
		"command": cty.StringVal("sleep 30; sleep 30"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	doneCh := make(chan struct{})
	startTime := time.Now()
	go func() {
		defer close(doneCh)
		// The functionality of p.Apply is tested in TestResourceProvider_Apply.
		// Because p.Apply is called in a goroutine, trying to t.Fatal() on its
		// result would be ignored or would cause a panic if the parent goroutine
		// has already completed.
		_ = p.ProvisionResource(provisioners.ProvisionResourceRequest{
			Config:   c,
			UIOutput: output,
		})
	}()

	mustExceed := (50 * time.Millisecond)
	select {
	case <-doneCh:
		t.Fatalf("expected to finish sometime after %s finished in %s", mustExceed, time.Since(startTime))
	case <-time.After(mustExceed):
		t.Logf("correctly took longer than %s", mustExceed)
	}

	// Stop it
	stopTime := time.Now()
	p.Stop()

	maxTempl := "expected to finish under %s, finished in %s"
	finishWithin := (2 * time.Second)
	select {
	case <-doneCh:
		t.Logf(maxTempl, finishWithin, time.Since(stopTime))
	case <-time.After(finishWithin):
		t.Fatalf(maxTempl, finishWithin, time.Since(stopTime))
	}
}

func TestResourceProvider_ApplyCustomInterpreter(t *testing.T) {
	output := cli.NewMockUi()
	p := New()

	schema := p.GetSchema().Provisioner

	c, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		"interpreter": cty.ListVal([]cty.Value{cty.StringVal("echo"), cty.StringVal("is")}),
		"command":     cty.StringVal("not really an interpreter"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	resp := p.ProvisionResource(provisioners.ProvisionResourceRequest{
		Config:   c,
		UIOutput: output,
	})

	if resp.Diagnostics.HasErrors() {
		t.Fatal(resp.Diagnostics.Err())
	}

	got := strings.TrimSpace(output.OutputWriter.String())
	want := `Executing: ["echo" "is" "not really an interpreter"]
is not really an interpreter`
	if got != want {
		t.Errorf("wrong output\ngot:  %s\nwant: %s", got, want)
	}
}

func TestResourceProvider_ApplyCustomWorkingDirectory(t *testing.T) {
	testdir := "working_dir_test"
	os.Mkdir(testdir, 0755)
	defer os.Remove(testdir)

	output := cli.NewMockUi()
	p := New()
	schema := p.GetSchema().Provisioner

	c, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		"working_dir": cty.StringVal(testdir),
		"command":     cty.StringVal("echo `pwd`"),
	}))
	if err != nil {
		t.Fatal(err)
	}

	resp := p.ProvisionResource(provisioners.ProvisionResourceRequest{
		Config:   c,
		UIOutput: output,
	})

	if resp.Diagnostics.HasErrors() {
		t.Fatal(resp.Diagnostics.Err())
	}

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	got := strings.TrimSpace(output.OutputWriter.String())
	want := "Executing: [\"/bin/sh\" \"-c\" \"echo `pwd`\"]\n" + dir + "/" + testdir
	if got != want {
		t.Errorf("wrong output\ngot:  %s\nwant: %s", got, want)
	}
}

func TestResourceProvider_ApplyCustomEnv(t *testing.T) {
	output := cli.NewMockUi()
	p := New()
	schema := p.GetSchema().Provisioner

	c, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		"command": cty.StringVal("echo $FOO $BAR $BAZ"),
		"environment": cty.MapVal(map[string]cty.Value{
			"FOO": cty.StringVal("BAR"),
			"BAR": cty.StringVal("1"),
			"BAZ": cty.StringVal("true"),
		}),
	}))
	if err != nil {
		t.Fatal(err)
	}

	resp := p.ProvisionResource(provisioners.ProvisionResourceRequest{
		Config:   c,
		UIOutput: output,
	})
	if resp.Diagnostics.HasErrors() {
		t.Fatal(resp.Diagnostics.Err())
	}

	got := strings.TrimSpace(output.OutputWriter.String())
	want := `Executing: ["/bin/sh" "-c" "echo $FOO $BAR $BAZ"]
BAR 1 true`
	if got != want {
		t.Errorf("wrong output\ngot:  %s\nwant: %s", got, want)
	}
}

// Validate that Stop can Close can be called even when not provisioning.
func TestResourceProvisioner_StopClose(t *testing.T) {
	p := New()
	p.Stop()
	p.Close()
}

func TestResourceProvisioner_nullsInOptionals(t *testing.T) {
	output := cli.NewMockUi()
	p := New()
	schema := p.GetSchema().Provisioner

	for i, cfg := range []cty.Value{
		cty.ObjectVal(map[string]cty.Value{
			"command": cty.StringVal("echo OK"),
			"environment": cty.MapVal(map[string]cty.Value{
				"FOO": cty.NullVal(cty.String),
			}),
		}),
		cty.ObjectVal(map[string]cty.Value{
			"command":     cty.StringVal("echo OK"),
			"environment": cty.NullVal(cty.Map(cty.String)),
		}),
		cty.ObjectVal(map[string]cty.Value{
			"command":     cty.StringVal("echo OK"),
			"interpreter": cty.ListVal([]cty.Value{cty.NullVal(cty.String)}),
		}),
		cty.ObjectVal(map[string]cty.Value{
			"command":     cty.StringVal("echo OK"),
			"interpreter": cty.NullVal(cty.List(cty.String)),
		}),
		cty.ObjectVal(map[string]cty.Value{
			"command":     cty.StringVal("echo OK"),
			"working_dir": cty.NullVal(cty.String),
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
