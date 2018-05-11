package terraform

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/config/configschema"
)

// debugInfo should be safe when nil
func TestDebugInfo_nil(t *testing.T) {
	var d *debugInfo

	d.SetPhase("none")
	d.WriteFile("none", nil)
	d.Close()
}

func TestDebugInfo_basicFile(t *testing.T) {
	var w bytes.Buffer
	debug, err := newDebugInfo("test-debug-info", &w)
	if err != nil {
		t.Fatal(err)
	}
	debug.SetPhase("test")

	fileData := map[string][]byte{
		"file1": []byte("file 1 data"),
		"file2": []byte("file 2 data"),
		"file3": []byte("file 3 data"),
	}

	for f, d := range fileData {
		err = debug.WriteFile(f, d)
		if err != nil {
			t.Fatal(err)
		}
	}

	err = debug.Close()
	if err != nil {
		t.Fatal(err)
	}

	gz, err := gzip.NewReader(&w)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		// get the filename part of the archived file
		name := regexp.MustCompile(`\w+$`).FindString(hdr.Name)
		data := fileData[name]

		delete(fileData, name)

		tarData, err := ioutil.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, tarData) {
			t.Fatalf("got '%s' for file '%s'", tarData, name)
		}
	}

	for k := range fileData {
		t.Fatalf("didn't find file %s", k)
	}
}

// Test that we get logs and graphs from a walk. We're not looking for anything
// specific, since the output is going to change in the near future.
func TestDebug_plan(t *testing.T) {
	var out bytes.Buffer
	d, err := newDebugInfo("test-debug-info", &out)
	if err != nil {
		t.Fatal(err)
	}
	// set the global debug value
	dbug = d

	// run a basic plan
	m := testModule(t, "plan-good")
	p := mockProviderWithResourceTypeSchema("aws_instance", &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"num": {
				Type:     cty.Number,
				Optional: true,
			},
			"foo": {
				Type:     cty.Number,
				Optional: true,
			},
		},
	})
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	_, diags := ctx.Plan()
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	err = CloseDebugInfo()
	if err != nil {
		t.Fatal(err)
	}

	gz, err := gzip.NewReader(&out)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)

	graphLogs := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		// record any file that contains data
		if hdr.Size > 0 {
			if strings.HasSuffix(hdr.Name, "graph.json") {
				graphLogs++
			}
		}
	}

	if graphLogs == 0 {
		t.Fatal("no json graphs")
	}
}

// verify that no hooks panic on nil input
func TestDebugHook_nilArgs(t *testing.T) {
	// make sure debug isn't nil, so the hooks try to execute
	var w bytes.Buffer
	var err error
	dbug, err = newDebugInfo("test-debug-info", &w)
	if err != nil {
		t.Fatal(err)
	}

	var h DebugHook
	h.PostApply(nil, nil, nil)
	h.PostDiff(nil, nil)
	h.PostImportState(nil, nil)
	h.PostProvision(nil, "", nil)
	h.PostProvisionResource(nil, nil)
	h.PostRefresh(nil, nil)
	h.PostStateUpdate(nil)
	h.PreApply(nil, nil, nil)
	h.PreDiff(nil, nil)
	h.PreImportState(nil, "")
	h.PreProvision(nil, "")
	h.PreProvisionResource(nil, nil)
	h.PreRefresh(nil, nil)
	h.ProvisionOutput(nil, "", "")
}
