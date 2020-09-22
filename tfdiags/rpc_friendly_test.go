package tfdiags

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/hashicorp/hcl/v2"
)

func TestDiagnosticsForRPC(t *testing.T) {
	var diags Diagnostics
	diags = diags.Append(fmt.Errorf("bad"))
	diags = diags.Append(SimpleWarning("less bad"))
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "bad bad bad",
		Detail:   "badily bad bad",
		Subject: &hcl.Range{
			Filename: "foo",
		},
		Context: &hcl.Range{
			Filename: "bar",
		},
	})

	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)

	rpcDiags := diags.ForRPC()
	err := enc.Encode(rpcDiags)
	if err != nil {
		t.Fatalf("error on Encode: %s", err)
	}

	var got Diagnostics
	err = dec.Decode(&got)
	if err != nil {
		t.Fatalf("error on Decode: %s", err)
	}

	want := Diagnostics{
		&rpcFriendlyDiag{
			Severity_: Error,
			Summary_:  "bad",
		},
		&rpcFriendlyDiag{
			Severity_: Warning,
			Summary_:  "less bad",
		},
		&rpcFriendlyDiag{
			Severity_: Error,
			Summary_:  "bad bad bad",
			Detail_:   "badily bad bad",
			Subject_: &SourceRange{
				Filename: "foo",
			},
			Context_: &SourceRange{
				Filename: "bar",
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong result\ngot: %swant: %s", spew.Sdump(got), spew.Sdump(want))
	}
}
