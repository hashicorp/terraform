package addrs

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestModuleInstanceMoveDestination(t *testing.T) {
	tests := []struct {
		DeclModule       string
		StmtFrom, StmtTo string
		Reciever         string
		WantMatch        bool
		WantResult       string
	}{
		{
			``,
			`module.foo`,
			`module.bar`,
			`module.foo`,
			true,
			`module.bar`,
		},
		{
			``,
			`module.foo`,
			`module.bar`,
			`module.foo[1]`,
			true,
			`module.bar[1]`,
		},
		{
			``,
			`module.foo`,
			`module.bar`,
			`module.foo["a"]`,
			true,
			`module.bar["a"]`,
		},
		{
			``,
			`module.foo`,
			`module.bar.module.foo`,
			`module.foo`,
			true,
			`module.bar.module.foo`,
		},
		{
			``,
			`module.foo.module.bar`,
			`module.bar`,
			`module.foo.module.bar`,
			true,
			`module.bar`,
		},
		{
			``,
			`module.foo[1]`,
			`module.foo[2]`,
			`module.foo[1]`,
			true,
			`module.foo[2]`,
		},
		{
			``,
			`module.foo[1]`,
			`module.foo`,
			`module.foo[1]`,
			true,
			`module.foo`,
		},
		{
			``,
			`module.foo`,
			`module.foo[1]`,
			`module.foo`,
			true,
			`module.foo[1]`,
		},
		{
			``,
			`module.foo`,
			`module.foo[1]`,
			`module.foo.module.bar`,
			true,
			`module.foo[1].module.bar`,
		},
		{
			``,
			`module.foo`,
			`module.foo[1]`,
			`module.foo.module.bar[0]`,
			true,
			`module.foo[1].module.bar[0]`,
		},
		{
			``,
			`module.foo`,
			`module.bar.module.foo`,
			`module.foo[0]`,
			true,
			`module.bar.module.foo[0]`,
		},
		{
			``,
			`module.foo.module.bar`,
			`module.bar`,
			`module.foo.module.bar[0]`,
			true,
			`module.bar[0]`,
		},
		{
			`foo`,
			`module.bar`,
			`module.baz`,
			`module.foo.module.bar`,
			true,
			`module.foo.module.baz`,
		},
		{
			`foo`,
			`module.bar`,
			`module.baz`,
			`module.foo[1].module.bar`,
			true,
			`module.foo[1].module.baz`,
		},
		{
			`foo`,
			`module.bar`,
			`module.bar[1]`,
			`module.foo[1].module.bar`,
			true,
			`module.foo[1].module.bar[1]`,
		},
		{
			``,
			`module.foo[1]`,
			`module.foo[2]`,
			`module.foo`,
			false, // the receiver has a non-matching instance key (NoKey)
			``,
		},
		{
			``,
			`module.foo[1]`,
			`module.foo[2]`,
			`module.foo[2]`,
			false, // the receiver is already the "to" address
			``,
		},
		{
			``,
			`module.foo`,
			`module.bar`,
			``,
			false, // the root module can never be moved
			``,
		},
		{
			`foo`,
			`module.bar`,
			`module.bar[1]`,
			`module.boz`,
			false, // the receiver is outside the declaration module
			``,
		},
		{
			`foo.bar`,
			`module.bar`,
			`module.bar[1]`,
			`module.boz`,
			false, // the receiver is outside the declaration module
			``,
		},
		{
			`foo.bar`,
			`module.a`,
			`module.b`,
			`module.boz`,
			false, // the receiver is outside the declaration module
			``,
		},
		{
			``,
			`module.a1.module.a2`,
			`module.b1.module.b2`,
			`module.c`,
			false, // the receiver is outside the declaration module
			``,
		},
		{
			``,
			`module.a1.module.a2[0]`,
			`module.b1.module.b2[1]`,
			`module.c`,
			false, // the receiver is outside the declaration module
			``,
		},
		{
			``,
			`module.a1.module.a2`,
			`module.b1.module.b2`,
			`module.a1.module.b2`,
			false, // the receiver is outside the declaration module
			``,
		},
		{
			``,
			`module.a1.module.a2`,
			`module.b1.module.b2`,
			`module.b1.module.a2`,
			false, // the receiver is outside the declaration module
			``,
		},
		{
			``,
			`module.a1.module.a2[0]`,
			`module.b1.module.b2[1]`,
			`module.a1.module.b2[0]`,
			false, // the receiver is outside the declaration module
			``,
		},
		{
			``,
			`foo_instance.bar`,
			`foo_instance.baz`,
			`module.foo`,
			false, // a resource address can never match a module instance
			``,
		},
	}

	for _, test := range tests {
		t.Run(
			fmt.Sprintf(
				"%s: %s to %s with %s",
				test.DeclModule,
				test.StmtFrom, test.StmtTo,
				test.Reciever,
			),
			func(t *testing.T) {

				parseStmtEP := func(t *testing.T, input string) *MoveEndpoint {
					t.Helper()

					traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(input), "", hcl.InitialPos)
					if hclDiags.HasErrors() {
						// We're not trying to test the HCL parser here, so any
						// failures at this point are likely to be bugs in the
						// test case itself.
						t.Fatalf("syntax error: %s", hclDiags.Error())
					}

					moveEp, diags := ParseMoveEndpoint(traversal)
					if diags.HasErrors() {
						t.Fatalf("unexpected error: %s", diags.Err().Error())
					}
					return moveEp
				}

				fromEPLocal := parseStmtEP(t, test.StmtFrom)
				toEPLocal := parseStmtEP(t, test.StmtTo)

				declModule := RootModule
				if test.DeclModule != "" {
					declModule = strings.Split(test.DeclModule, ".")
				}
				fromEP, toEP := UnifyMoveEndpoints(declModule, fromEPLocal, toEPLocal)
				if fromEP == nil || toEP == nil {
					t.Fatalf("invalid test case: non-unifyable endpoints\nfrom: %s\nto:   %s", fromEPLocal, toEPLocal)
				}

				receiverAddr := RootModuleInstance
				if test.Reciever != "" {
					var diags tfdiags.Diagnostics
					receiverAddr, diags = ParseModuleInstanceStr(test.Reciever)
					if diags.HasErrors() {
						t.Fatalf("invalid reciever address: %s", diags.Err().Error())
					}
				}
				gotAddr, gotMatch := receiverAddr.MoveDestination(fromEP, toEP)
				if !test.WantMatch {
					if gotMatch {
						t.Errorf("unexpected match\nreciever: %s\nfrom:     %s\nto:       %s\nresult:   %s", test.Reciever, fromEP, toEP, gotAddr)
					}
					return
				}

				if !gotMatch {
					t.Errorf("unexpected non-match\nreciever: %s\nfrom:     %s\nto:       %s", test.Reciever, fromEP, toEP)
				}

				if gotStr, wantStr := gotAddr.String(), test.WantResult; gotStr != wantStr {
					t.Errorf("wrong result\ngot:  %s\nwant: %s", gotStr, wantStr)
				}
			},
		)
	}
}

func TestAbsResourceInstanceMoveDestination(t *testing.T) {
	tests := []struct {
		DeclModule       string
		StmtFrom, StmtTo string
		Reciever         string
		WantMatch        bool
		WantResult       string
	}{
		{
			``,
			`module.foo`,
			`module.bar`,
			`module.foo.test_object.beep`,
			true,
			`module.bar.test_object.beep`,
		},
		{
			``,
			`module.foo`,
			`module.bar`,
			`module.foo[1].test_object.beep`,
			true,
			`module.bar[1].test_object.beep`,
		},
		{
			``,
			`module.foo`,
			`module.bar`,
			`module.foo["a"].test_object.beep`,
			true,
			`module.bar["a"].test_object.beep`,
		},
		{
			``,
			`module.foo`,
			`module.bar.module.foo`,
			`module.foo.test_object.beep`,
			true,
			`module.bar.module.foo.test_object.beep`,
		},
		{
			``,
			`module.foo.module.bar`,
			`module.bar`,
			`module.foo.module.bar.test_object.beep`,
			true,
			`module.bar.test_object.beep`,
		},
		{
			``,
			`module.foo[1]`,
			`module.foo[2]`,
			`module.foo[1].test_object.beep`,
			true,
			`module.foo[2].test_object.beep`,
		},
		{
			``,
			`module.foo[1]`,
			`module.foo`,
			`module.foo[1].test_object.beep`,
			true,
			`module.foo.test_object.beep`,
		},
		{
			``,
			`module.foo`,
			`module.foo[1]`,
			`module.foo.test_object.beep`,
			true,
			`module.foo[1].test_object.beep`,
		},
		{
			``,
			`module.foo`,
			`module.foo[1]`,
			`module.foo.module.bar.test_object.beep`,
			true,
			`module.foo[1].module.bar.test_object.beep`,
		},
		{
			``,
			`module.foo`,
			`module.foo[1]`,
			`module.foo.module.bar[0].test_object.beep`,
			true,
			`module.foo[1].module.bar[0].test_object.beep`,
		},
		{
			``,
			`module.foo`,
			`module.bar.module.foo`,
			`module.foo[0].test_object.beep`,
			true,
			`module.bar.module.foo[0].test_object.beep`,
		},
		{
			``,
			`module.foo.module.bar`,
			`module.bar`,
			`module.foo.module.bar[0].test_object.beep`,
			true,
			`module.bar[0].test_object.beep`,
		},
		{
			`foo`,
			`module.bar`,
			`module.baz`,
			`module.foo.module.bar.test_object.beep`,
			true,
			`module.foo.module.baz.test_object.beep`,
		},
		{
			`foo`,
			`module.bar`,
			`module.baz`,
			`module.foo[1].module.bar.test_object.beep`,
			true,
			`module.foo[1].module.baz.test_object.beep`,
		},
		{
			`foo`,
			`module.bar`,
			`module.bar[1]`,
			`module.foo[1].module.bar.test_object.beep`,
			true,
			`module.foo[1].module.bar[1].test_object.beep`,
		},
		{
			``,
			`module.foo[1]`,
			`module.foo[2]`,
			`module.foo.test_object.beep`,
			false, // the receiver module has a non-matching instance key (NoKey)
			``,
		},
		{
			``,
			`module.foo[1]`,
			`module.foo[2]`,
			`module.foo[2].test_object.beep`,
			false, // the receiver is already at the "to" address
			``,
		},
		{
			`foo`,
			`module.bar`,
			`module.bar[1]`,
			`module.boz.test_object.beep`,
			false, // the receiver module is outside the declaration module
			``,
		},
		{
			`foo.bar`,
			`module.bar`,
			`module.bar[1]`,
			`module.boz.test_object.beep`,
			false, // the receiver module is outside the declaration module
			``,
		},
		{
			`foo.bar`,
			`module.a`,
			`module.b`,
			`module.boz.test_object.beep`,
			false, // the receiver module is outside the declaration module
			``,
		},
		{
			``,
			`module.a1.module.a2`,
			`module.b1.module.b2`,
			`module.c.test_object.beep`,
			false, // the receiver module is outside the declaration module
			``,
		},
		{
			``,
			`module.a1.module.a2[0]`,
			`module.b1.module.b2[1]`,
			`module.c.test_object.beep`,
			false, // the receiver module is outside the declaration module
			``,
		},
		{
			``,
			`module.a1.module.a2`,
			`module.b1.module.b2`,
			`module.a1.module.b2.test_object.beep`,
			false, // the receiver module is outside the declaration module
			``,
		},
		{
			``,
			`module.a1.module.a2`,
			`module.b1.module.b2`,
			`module.b1.module.a2.test_object.beep`,
			false, // the receiver module is outside the declaration module
			``,
		},
		{
			``,
			`module.a1.module.a2[0]`,
			`module.b1.module.b2[1]`,
			`module.a1.module.b2[0].test_object.beep`,
			false, // the receiver module is outside the declaration module
			``,
		},
		{
			``,
			`foo_instance.bar`,
			`foo_instance.baz`,
			`module.foo.test_object.beep`,
			false, // the resource address is unrelated to the move statements
			``,
		},
	}

	for _, test := range tests {
		t.Run(
			fmt.Sprintf(
				"%s: %s to %s with %s",
				test.DeclModule,
				test.StmtFrom, test.StmtTo,
				test.Reciever,
			),
			func(t *testing.T) {

				parseStmtEP := func(t *testing.T, input string) *MoveEndpoint {
					t.Helper()

					traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(input), "", hcl.InitialPos)
					if hclDiags.HasErrors() {
						// We're not trying to test the HCL parser here, so any
						// failures at this point are likely to be bugs in the
						// test case itself.
						t.Fatalf("syntax error: %s", hclDiags.Error())
					}

					moveEp, diags := ParseMoveEndpoint(traversal)
					if diags.HasErrors() {
						t.Fatalf("unexpected error: %s", diags.Err().Error())
					}
					return moveEp
				}

				fromEPLocal := parseStmtEP(t, test.StmtFrom)
				toEPLocal := parseStmtEP(t, test.StmtTo)

				declModule := RootModule
				if test.DeclModule != "" {
					declModule = strings.Split(test.DeclModule, ".")
				}
				fromEP, toEP := UnifyMoveEndpoints(declModule, fromEPLocal, toEPLocal)
				if fromEP == nil || toEP == nil {
					t.Fatalf("invalid test case: non-unifyable endpoints\nfrom: %s\nto:   %s", fromEPLocal, toEPLocal)
				}

				receiverAddr, diags := ParseAbsResourceInstanceStr(test.Reciever)
				if diags.HasErrors() {
					t.Fatalf("invalid reciever address: %s", diags.Err().Error())
				}
				gotAddr, gotMatch := receiverAddr.MoveDestination(fromEP, toEP)
				if !test.WantMatch {
					if gotMatch {
						t.Errorf("unexpected match\nreciever: %s\nfrom:     %s\nto:       %s\nresult:   %s", test.Reciever, fromEP, toEP, gotAddr)
					}
					return
				}

				if !gotMatch {
					t.Errorf("unexpected non-match\nreciever: %s\nfrom:     %s\nto:       %s", test.Reciever, fromEP, toEP)
				}

				if gotStr, wantStr := gotAddr.String(), test.WantResult; gotStr != wantStr {
					t.Errorf("wrong result\ngot:  %s\nwant: %s", gotStr, wantStr)
				}
			},
		)
	}
}

func TestAbsResourceMoveDestination(t *testing.T) {
	tests := []struct {
		DeclModule       string
		StmtFrom, StmtTo string
		Reciever         string
		WantMatch        bool
		WantResult       string
	}{
		{
			``,
			`module.foo`,
			`module.bar`,
			`module.foo.test_object.beep`,
			true,
			`module.bar.test_object.beep`,
		},
		{
			``,
			`module.foo`,
			`module.bar`,
			`module.foo[1].test_object.beep`,
			true,
			`module.bar[1].test_object.beep`,
		},
		{
			``,
			`module.foo`,
			`module.bar`,
			`module.foo["a"].test_object.beep`,
			true,
			`module.bar["a"].test_object.beep`,
		},
		{
			``,
			`module.foo`,
			`module.bar.module.foo`,
			`module.foo.test_object.beep`,
			true,
			`module.bar.module.foo.test_object.beep`,
		},
		{
			``,
			`module.foo.module.bar`,
			`module.bar`,
			`module.foo.module.bar.test_object.beep`,
			true,
			`module.bar.test_object.beep`,
		},
		{
			``,
			`module.foo[1]`,
			`module.foo[2]`,
			`module.foo[1].test_object.beep`,
			true,
			`module.foo[2].test_object.beep`,
		},
		{
			``,
			`module.foo[1]`,
			`module.foo`,
			`module.foo[1].test_object.beep`,
			true,
			`module.foo.test_object.beep`,
		},
		{
			``,
			`module.foo`,
			`module.foo[1]`,
			`module.foo.test_object.beep`,
			true,
			`module.foo[1].test_object.beep`,
		},
		{
			``,
			`module.foo`,
			`module.foo[1]`,
			`module.foo.module.bar.test_object.beep`,
			true,
			`module.foo[1].module.bar.test_object.beep`,
		},
		{
			``,
			`module.foo`,
			`module.foo[1]`,
			`module.foo.module.bar[0].test_object.beep`,
			true,
			`module.foo[1].module.bar[0].test_object.beep`,
		},
		{
			``,
			`module.foo`,
			`module.bar.module.foo`,
			`module.foo[0].test_object.beep`,
			true,
			`module.bar.module.foo[0].test_object.beep`,
		},
		{
			``,
			`module.foo.module.bar`,
			`module.bar`,
			`module.foo.module.bar[0].test_object.beep`,
			true,
			`module.bar[0].test_object.beep`,
		},
		{
			`foo`,
			`module.bar`,
			`module.baz`,
			`module.foo.module.bar.test_object.beep`,
			true,
			`module.foo.module.baz.test_object.beep`,
		},
		{
			`foo`,
			`module.bar`,
			`module.baz`,
			`module.foo[1].module.bar.test_object.beep`,
			true,
			`module.foo[1].module.baz.test_object.beep`,
		},
		{
			`foo`,
			`module.bar`,
			`module.bar[1]`,
			`module.foo[1].module.bar.test_object.beep`,
			true,
			`module.foo[1].module.bar[1].test_object.beep`,
		},
		{
			``,
			`module.foo[1]`,
			`module.foo[2]`,
			`module.foo.test_object.beep`,
			false, // the receiver module has a non-matching instance key (NoKey)
			``,
		},
		{
			``,
			`module.foo[1]`,
			`module.foo[2]`,
			`module.foo[2].test_object.beep`,
			false, // the receiver is already at the "to" address
			``,
		},
		{
			`foo`,
			`module.bar`,
			`module.bar[1]`,
			`module.boz.test_object.beep`,
			false, // the receiver module is outside the declaration module
			``,
		},
		{
			`foo.bar`,
			`module.bar`,
			`module.bar[1]`,
			`module.boz.test_object.beep`,
			false, // the receiver module is outside the declaration module
			``,
		},
		{
			`foo.bar`,
			`module.a`,
			`module.b`,
			`module.boz.test_object.beep`,
			false, // the receiver module is outside the declaration module
			``,
		},
		{
			``,
			`module.a1.module.a2`,
			`module.b1.module.b2`,
			`module.c.test_object.beep`,
			false, // the receiver module is outside the declaration module
			``,
		},
		{
			``,
			`module.a1.module.a2[0]`,
			`module.b1.module.b2[1]`,
			`module.c.test_object.beep`,
			false, // the receiver module is outside the declaration module
			``,
		},
		{
			``,
			`module.a1.module.a2`,
			`module.b1.module.b2`,
			`module.a1.module.b2.test_object.beep`,
			false, // the receiver module is outside the declaration module
			``,
		},
		{
			``,
			`module.a1.module.a2`,
			`module.b1.module.b2`,
			`module.b1.module.a2.test_object.beep`,
			false, // the receiver module is outside the declaration module
			``,
		},
		{
			``,
			`module.a1.module.a2[0]`,
			`module.b1.module.b2[1]`,
			`module.a1.module.b2[0].test_object.beep`,
			false, // the receiver module is outside the declaration module
			``,
		},
		{
			``,
			`foo_instance.bar`,
			`foo_instance.baz`,
			`module.foo.test_object.beep`,
			false, // the resource address is unrelated to the move statements
			``,
		},
	}

	for _, test := range tests {
		t.Run(
			fmt.Sprintf(
				"%s: %s to %s with %s",
				test.DeclModule,
				test.StmtFrom, test.StmtTo,
				test.Reciever,
			),
			func(t *testing.T) {

				parseStmtEP := func(t *testing.T, input string) *MoveEndpoint {
					t.Helper()

					traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(input), "", hcl.InitialPos)
					if hclDiags.HasErrors() {
						// We're not trying to test the HCL parser here, so any
						// failures at this point are likely to be bugs in the
						// test case itself.
						t.Fatalf("syntax error: %s", hclDiags.Error())
					}

					moveEp, diags := ParseMoveEndpoint(traversal)
					if diags.HasErrors() {
						t.Fatalf("unexpected error: %s", diags.Err().Error())
					}
					return moveEp
				}

				fromEPLocal := parseStmtEP(t, test.StmtFrom)
				toEPLocal := parseStmtEP(t, test.StmtTo)

				declModule := RootModule
				if test.DeclModule != "" {
					declModule = strings.Split(test.DeclModule, ".")
				}
				fromEP, toEP := UnifyMoveEndpoints(declModule, fromEPLocal, toEPLocal)
				if fromEP == nil || toEP == nil {
					t.Fatalf("invalid test case: non-unifyable endpoints\nfrom: %s\nto:   %s", fromEPLocal, toEPLocal)
				}

				// We only have an AbsResourceInstance parser, not an
				// AbsResourceParser, and so we'll just cheat and parse this
				// as a resource instance but fail if it includes an instance
				// key.
				receiverInstanceAddr, diags := ParseAbsResourceInstanceStr(test.Reciever)
				if diags.HasErrors() {
					t.Fatalf("invalid reciever address: %s", diags.Err().Error())
				}
				if receiverInstanceAddr.Resource.Key != NoKey {
					t.Fatalf("invalid reciever address: must be a resource, not a resource instance")
				}
				receiverAddr := receiverInstanceAddr.ContainingResource()
				gotAddr, gotMatch := receiverAddr.MoveDestination(fromEP, toEP)
				if !test.WantMatch {
					if gotMatch {
						t.Errorf("unexpected match\nreciever: %s\nfrom:     %s\nto:       %s\nresult:   %s", test.Reciever, fromEP, toEP, gotAddr)
					}
					return
				}

				if !gotMatch {
					t.Errorf("unexpected non-match\nreciever: %s\nfrom:     %s\nto:       %s", test.Reciever, fromEP, toEP)
				}

				if gotStr, wantStr := gotAddr.String(), test.WantResult; gotStr != wantStr {
					t.Errorf("wrong result\ngot:  %s\nwant: %s", gotStr, wantStr)
				}
			},
		)
	}
}
