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
		Receiver         string
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
				test.Receiver,
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
				if test.Receiver != "" {
					var diags tfdiags.Diagnostics
					receiverAddr, diags = ParseModuleInstanceStr(test.Receiver)
					if diags.HasErrors() {
						t.Fatalf("invalid reciever address: %s", diags.Err().Error())
					}
				}
				gotAddr, gotMatch := receiverAddr.MoveDestination(fromEP, toEP)
				if !test.WantMatch {
					if gotMatch {
						t.Errorf("unexpected match\nreceiver: %s\nfrom:     %s\nto:       %s\nresult:   %s", test.Receiver, fromEP, toEP, gotAddr)
					}
					return
				}

				if !gotMatch {
					t.Errorf("unexpected non-match\nreceiver: %s\nfrom:     %s\nto:       %s", test.Receiver, fromEP, toEP)
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
		Receiver         string
		WantMatch        bool
		WantResult       string
	}{
		{
			``,
			`test_object.beep`,
			`test_object.boop`,
			`test_object.beep`,
			true,
			`test_object.boop`,
		},
		{
			``,
			`test_object.beep`,
			`test_object.beep[2]`,
			`test_object.beep`,
			true,
			`test_object.beep[2]`,
		},
		{
			``,
			`test_object.beep`,
			`module.foo.test_object.beep`,
			`test_object.beep`,
			true,
			`module.foo.test_object.beep`,
		},
		{
			``,
			`test_object.beep[2]`,
			`module.foo.test_object.beep["a"]`,
			`test_object.beep[2]`,
			true,
			`module.foo.test_object.beep["a"]`,
		},
		{
			``,
			`test_object.beep`,
			`module.foo[0].test_object.beep`,
			`test_object.beep`,
			true,
			`module.foo[0].test_object.beep`,
		},
		{
			``,
			`module.foo.test_object.beep`,
			`test_object.beep`,
			`module.foo.test_object.beep`,
			true,
			`test_object.beep`,
		},
		{
			``,
			`module.foo[0].test_object.beep`,
			`test_object.beep`,
			`module.foo[0].test_object.beep`,
			true,
			`test_object.beep`,
		},
		{
			`foo`,
			`test_object.beep`,
			`test_object.boop`,
			`module.foo[0].test_object.beep`,
			true,
			`module.foo[0].test_object.boop`,
		},
		{
			`foo`,
			`test_object.beep`,
			`test_object.beep[1]`,
			`module.foo[0].test_object.beep`,
			true,
			`module.foo[0].test_object.beep[1]`,
		},
		{
			``,
			`test_object.beep`,
			`test_object.boop`,
			`test_object.boop`,
			false, // the reciever is already the "to" address
			``,
		},
		{
			``,
			`test_object.beep[1]`,
			`test_object.beep[2]`,
			`test_object.beep[5]`,
			false, // the receiver has a non-matching instance key
			``,
		},
		{
			`foo`,
			`test_object.beep`,
			`test_object.boop`,
			`test_object.beep`,
			false, // the receiver is not inside an instance of module "foo"
			``,
		},
		{
			`foo.bar`,
			`test_object.beep`,
			`test_object.boop`,
			`test_object.beep`,
			false, // the receiver is not inside an instance of module "foo.bar"
			``,
		},
		{
			``,
			`module.foo[0].test_object.beep`,
			`test_object.beep`,
			`module.foo[1].test_object.beep`,
			false, // receiver is in a different instance of module.foo
			``,
		},

		// Moving a module also moves all of the resources declared within it.
		// The following tests all cover variations of that rule.
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
				test.Receiver,
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

				receiverAddr, diags := ParseAbsResourceInstanceStr(test.Receiver)
				if diags.HasErrors() {
					t.Fatalf("invalid reciever address: %s", diags.Err().Error())
				}
				gotAddr, gotMatch := receiverAddr.MoveDestination(fromEP, toEP)
				if !test.WantMatch {
					if gotMatch {
						t.Errorf("unexpected match\nreceiver: %s\nfrom:     %s\nto:       %s\nresult:   %s", test.Receiver, fromEP, toEP, gotAddr)
					}
					return
				}

				if !gotMatch {
					t.Fatalf("unexpected non-match\nreceiver: %s (%T)\nfrom:     %s\nto:       %s\ngot:      (no match)\nwant:     %s", test.Receiver, receiverAddr, fromEP, toEP, test.WantResult)
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
		Receiver         string
		WantMatch        bool
		WantResult       string
	}{
		{
			``,
			`test_object.beep`,
			`test_object.boop`,
			`test_object.beep`,
			true,
			`test_object.boop`,
		},
		{
			``,
			`test_object.beep`,
			`module.foo.test_object.beep`,
			`test_object.beep`,
			true,
			`module.foo.test_object.beep`,
		},
		{
			``,
			`test_object.beep`,
			`module.foo[0].test_object.beep`,
			`test_object.beep`,
			true,
			`module.foo[0].test_object.beep`,
		},
		{
			``,
			`module.foo.test_object.beep`,
			`test_object.beep`,
			`module.foo.test_object.beep`,
			true,
			`test_object.beep`,
		},
		{
			``,
			`module.foo[0].test_object.beep`,
			`test_object.beep`,
			`module.foo[0].test_object.beep`,
			true,
			`test_object.beep`,
		},
		{
			`foo`,
			`test_object.beep`,
			`test_object.boop`,
			`module.foo[0].test_object.beep`,
			true,
			`module.foo[0].test_object.boop`,
		},
		{
			``,
			`test_object.beep`,
			`test_object.boop`,
			`test_object.boop`,
			false, // the reciever is already the "to" address
			``,
		},
		{
			`foo`,
			`test_object.beep`,
			`test_object.boop`,
			`test_object.beep`,
			false, // the receiver is not inside an instance of module "foo"
			``,
		},
		{
			`foo.bar`,
			`test_object.beep`,
			`test_object.boop`,
			`test_object.beep`,
			false, // the receiver is not inside an instance of module "foo.bar"
			``,
		},
		{
			``,
			`module.foo[0].test_object.beep`,
			`test_object.beep`,
			`module.foo[1].test_object.beep`,
			false, // receiver is in a different instance of module.foo
			``,
		},

		// Moving a module also moves all of the resources declared within it.
		// The following tests all cover variations of that rule.
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

	for i, test := range tests {
		t.Run(
			fmt.Sprintf(
				"[%02d] %s: %s to %s with %s",
				i,
				test.DeclModule,
				test.StmtFrom, test.StmtTo,
				test.Receiver,
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
				receiverInstanceAddr, diags := ParseAbsResourceInstanceStr(test.Receiver)
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
						t.Errorf("unexpected match\nreceiver: %s (%T)\nfrom:     %s\nto:       %s\nresult:   %s", test.Receiver, receiverAddr, fromEP, toEP, gotAddr)
					}
					return
				}

				if !gotMatch {
					t.Fatalf("unexpected non-match\nreceiver: %s (%T)\nfrom:     %s\nto:       %s\ngot:      no match\nwant:     %s", test.Receiver, receiverAddr, fromEP, toEP, test.WantResult)
				}

				if gotStr, wantStr := gotAddr.String(), test.WantResult; gotStr != wantStr {
					t.Errorf("wrong result\ngot:  %s\nwant: %s", gotStr, wantStr)
				}
			},
		)
	}
}

func TestMoveEndpointChainAndNested(t *testing.T) {
	tests := []struct {
		Endpoint, Other            AbsMoveable
		EndpointMod, OtherMod      Module
		CanChainFrom, NestedWithin bool
	}{
		{
			Endpoint: AbsModuleCall{
				Module: mustParseModuleInstanceStr("module.foo[2]"),
				Call:   ModuleCall{Name: "bar"},
			},
			Other: AbsModuleCall{
				Module: mustParseModuleInstanceStr("module.foo[2]"),
				Call:   ModuleCall{Name: "bar"},
			},
			CanChainFrom: true,
			NestedWithin: false,
		},

		{
			Endpoint: mustParseModuleInstanceStr("module.foo[2]"),
			Other: AbsModuleCall{
				Module: mustParseModuleInstanceStr("module.foo[2]"),
				Call:   ModuleCall{Name: "bar"},
			},
			CanChainFrom: false,
			NestedWithin: false,
		},

		{
			Endpoint: mustParseModuleInstanceStr("module.foo[2].module.bar[2]"),
			Other: AbsModuleCall{
				Module: RootModuleInstance,
				Call:   ModuleCall{Name: "foo"},
			},
			CanChainFrom: false,
			NestedWithin: true,
		},

		{
			Endpoint: mustParseAbsResourceInstanceStr("module.foo[2].module.bar.resource.baz").ContainingResource(),
			Other: AbsModuleCall{
				Module: mustParseModuleInstanceStr("module.foo[2]"),
				Call:   ModuleCall{Name: "bar"},
			},
			CanChainFrom: false,
			NestedWithin: true,
		},

		{
			Endpoint: mustParseAbsResourceInstanceStr("module.foo[2].module.bar[3].resource.baz[2]"),
			Other: AbsModuleCall{
				Module: mustParseModuleInstanceStr("module.foo[2]"),
				Call:   ModuleCall{Name: "bar"},
			},
			CanChainFrom: false,
			NestedWithin: true,
		},

		{
			Endpoint: AbsModuleCall{
				Module: mustParseModuleInstanceStr("module.foo[2]"),
				Call:   ModuleCall{Name: "bar"},
			},
			Other:        mustParseModuleInstanceStr("module.foo[2]"),
			CanChainFrom: false,
			NestedWithin: true,
		},

		{
			Endpoint:     mustParseModuleInstanceStr("module.foo[2]"),
			Other:        mustParseModuleInstanceStr("module.foo[2]"),
			CanChainFrom: true,
			NestedWithin: false,
		},

		{
			Endpoint:     mustParseAbsResourceInstanceStr("module.foo[2].resource.baz").ContainingResource(),
			Other:        mustParseModuleInstanceStr("module.foo[2]"),
			CanChainFrom: false,
			NestedWithin: true,
		},

		{
			Endpoint:     mustParseAbsResourceInstanceStr("module.foo[2].module.bar.resource.baz"),
			Other:        mustParseModuleInstanceStr("module.foo[2]"),
			CanChainFrom: false,
			NestedWithin: true,
		},

		{
			Endpoint: AbsModuleCall{
				Module: mustParseModuleInstanceStr("module.foo[2]"),
				Call:   ModuleCall{Name: "bar"},
			},
			Other:        mustParseAbsResourceInstanceStr("module.foo[2].resource.baz").ContainingResource(),
			CanChainFrom: false,
			NestedWithin: false,
		},

		{
			Endpoint:     mustParseModuleInstanceStr("module.foo[2]"),
			Other:        mustParseAbsResourceInstanceStr("module.foo[2].resource.baz").ContainingResource(),
			CanChainFrom: false,
			NestedWithin: false,
		},

		{
			Endpoint:     mustParseAbsResourceInstanceStr("module.foo[2].resource.baz").ContainingResource(),
			Other:        mustParseAbsResourceInstanceStr("module.foo[2].resource.baz").ContainingResource(),
			CanChainFrom: true,
			NestedWithin: false,
		},

		{
			Endpoint:     mustParseAbsResourceInstanceStr("module.foo[2].resource.baz"),
			Other:        mustParseAbsResourceInstanceStr("module.foo[2].resource.baz[2]").ContainingResource(),
			CanChainFrom: false,
			NestedWithin: true,
		},

		{
			Endpoint: AbsModuleCall{
				Module: mustParseModuleInstanceStr("module.foo[2]"),
				Call:   ModuleCall{Name: "bar"},
			},
			Other:        mustParseAbsResourceInstanceStr("module.foo[2].resource.baz"),
			CanChainFrom: false,
		},

		{
			Endpoint:     mustParseModuleInstanceStr("module.foo[2]"),
			Other:        mustParseAbsResourceInstanceStr("module.foo[2].resource.baz"),
			CanChainFrom: false,
		},
		{
			Endpoint:     mustParseAbsResourceInstanceStr("module.foo[2].resource.baz").ContainingResource(),
			Other:        mustParseAbsResourceInstanceStr("module.foo[2].resource.baz"),
			CanChainFrom: false,
		},

		{
			Endpoint:     mustParseAbsResourceInstanceStr("module.foo[2].resource.baz"),
			Other:        mustParseAbsResourceInstanceStr("module.foo[2].resource.baz"),
			CanChainFrom: true,
		},

		{
			Endpoint:     mustParseAbsResourceInstanceStr("resource.baz"),
			EndpointMod:  Module{"foo"},
			Other:        mustParseAbsResourceInstanceStr("module.foo[2].resource.baz"),
			CanChainFrom: true,
		},

		{
			Endpoint:     mustParseAbsResourceInstanceStr("module.foo[2].resource.baz"),
			Other:        mustParseAbsResourceInstanceStr("resource.baz"),
			OtherMod:     Module{"foo"},
			CanChainFrom: true,
		},

		{
			Endpoint:     mustParseAbsResourceInstanceStr("resource.baz"),
			EndpointMod:  Module{"foo"},
			Other:        mustParseAbsResourceInstanceStr("resource.baz"),
			OtherMod:     Module{"foo"},
			CanChainFrom: true,
		},

		{
			Endpoint:     mustParseAbsResourceInstanceStr("resource.baz").ContainingResource(),
			EndpointMod:  Module{"foo"},
			Other:        mustParseAbsResourceInstanceStr("module.foo[2].resource.baz").ContainingResource(),
			CanChainFrom: true,
		},

		{
			Endpoint:     mustParseModuleInstanceStr("module.foo[2].module.baz"),
			Other:        mustParseModuleInstanceStr("module.baz"),
			OtherMod:     Module{"foo"},
			CanChainFrom: true,
		},

		{
			Endpoint: AbsModuleCall{
				Call: ModuleCall{Name: "bing"},
			},
			EndpointMod: Module{"foo", "baz"},
			Other: AbsModuleCall{
				Module: mustParseModuleInstanceStr("module.baz"),
				Call:   ModuleCall{Name: "bing"},
			},
			OtherMod:     Module{"foo"},
			CanChainFrom: true,
		},

		{
			Endpoint:     mustParseAbsResourceInstanceStr("resource.baz"),
			EndpointMod:  Module{"foo"},
			Other:        mustParseAbsResourceInstanceStr("module.foo[2].resource.baz").ContainingResource(),
			NestedWithin: true,
		},

		{
			Endpoint:     mustParseAbsResourceInstanceStr("module.foo[2].resource.baz"),
			Other:        mustParseAbsResourceInstanceStr("resource.baz").ContainingResource(),
			OtherMod:     Module{"foo"},
			NestedWithin: true,
		},

		{
			Endpoint:     mustParseAbsResourceInstanceStr("resource.baz"),
			EndpointMod:  Module{"foo"},
			Other:        mustParseAbsResourceInstanceStr("resource.baz").ContainingResource(),
			OtherMod:     Module{"foo"},
			NestedWithin: true,
		},

		{
			Endpoint:     mustParseAbsResourceInstanceStr("ressurce.baz").ContainingResource(),
			EndpointMod:  Module{"foo"},
			Other:        mustParseModuleInstanceStr("module.foo[2]"),
			NestedWithin: true,
		},

		{
			Endpoint: AbsModuleCall{
				Call: ModuleCall{Name: "bang"},
			},
			EndpointMod: Module{"foo", "baz", "bing"},
			Other: AbsModuleCall{
				Module: mustParseModuleInstanceStr("module.baz"),
				Call:   ModuleCall{Name: "bing"},
			},
			OtherMod:     Module{"foo"},
			NestedWithin: true,
		},

		{
			Endpoint: AbsModuleCall{
				Module: mustParseModuleInstanceStr("module.bing"),
				Call:   ModuleCall{Name: "bang"},
			},
			EndpointMod: Module{"foo", "baz"},
			Other: AbsModuleCall{
				Module: mustParseModuleInstanceStr("module.foo.module.baz"),
				Call:   ModuleCall{Name: "bing"},
			},
			NestedWithin: true,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("[%02d]%s.CanChainFrom(%s)", i, test.Endpoint, test.Other),
			func(t *testing.T) {
				endpoint := &MoveEndpointInModule{
					relSubject: test.Endpoint,
					module:     test.EndpointMod,
				}

				other := &MoveEndpointInModule{
					relSubject: test.Other,
					module:     test.OtherMod,
				}

				if endpoint.CanChainFrom(other) != test.CanChainFrom {
					t.Errorf("expected %s CanChainFrom %s == %t", endpoint, other, test.CanChainFrom)
				}

				if endpoint.NestedWithin(other) != test.NestedWithin {
					t.Errorf("expected %s NestedWithin %s == %t", endpoint, other, test.NestedWithin)
				}
			},
		)
	}
}

func TestSelectsModule(t *testing.T) {
	tests := []struct {
		Endpoint *MoveEndpointInModule
		Addr     ModuleInstance
		Selects  bool
	}{
		{
			Endpoint: &MoveEndpointInModule{
				relSubject: AbsModuleCall{
					Module: mustParseModuleInstanceStr("module.foo[2]"),
					Call:   ModuleCall{Name: "bar"},
				},
			},
			Addr:    mustParseModuleInstanceStr("module.foo[2].module.bar[1]"),
			Selects: true,
		},
		{
			Endpoint: &MoveEndpointInModule{
				module: mustParseModuleInstanceStr("module.foo").Module(),
				relSubject: AbsModuleCall{
					Module: mustParseModuleInstanceStr("module.bar[2]"),
					Call:   ModuleCall{Name: "baz"},
				},
			},
			Addr:    mustParseModuleInstanceStr("module.foo[2].module.bar[2].module.baz"),
			Selects: true,
		},
		{
			Endpoint: &MoveEndpointInModule{
				module: mustParseModuleInstanceStr("module.foo").Module(),
				relSubject: AbsModuleCall{
					Module: mustParseModuleInstanceStr("module.bar[2]"),
					Call:   ModuleCall{Name: "baz"},
				},
			},
			Addr:    mustParseModuleInstanceStr("module.foo[2].module.bar[1].module.baz"),
			Selects: false,
		},
		{
			Endpoint: &MoveEndpointInModule{
				relSubject: AbsModuleCall{
					Module: mustParseModuleInstanceStr("module.bar"),
					Call:   ModuleCall{Name: "baz"},
				},
			},
			Addr:    mustParseModuleInstanceStr("module.bar[1].module.baz"),
			Selects: false,
		},
		{
			Endpoint: &MoveEndpointInModule{
				module:     mustParseModuleInstanceStr("module.foo").Module(),
				relSubject: mustParseAbsResourceInstanceStr(`module.bar.resource.name["key"]`),
			},
			Addr:    mustParseModuleInstanceStr(`module.foo[1].module.bar`),
			Selects: true,
		},
		{
			Endpoint: &MoveEndpointInModule{
				relSubject: mustParseModuleInstanceStr(`module.bar.module.baz["key"]`),
			},
			Addr:    mustParseModuleInstanceStr(`module.bar.module.baz["key"]`),
			Selects: true,
		},
		{
			Endpoint: &MoveEndpointInModule{
				relSubject: mustParseAbsResourceInstanceStr(`module.bar.module.baz["key"].resource.name`).ContainingResource(),
			},
			Addr:    mustParseModuleInstanceStr(`module.bar.module.baz["key"]`),
			Selects: true,
		},
		{
			Endpoint: &MoveEndpointInModule{
				module:     mustParseModuleInstanceStr("module.nope").Module(),
				relSubject: mustParseAbsResourceInstanceStr(`module.bar.resource.name["key"]`),
			},
			Addr:    mustParseModuleInstanceStr(`module.foo[1].module.bar`),
			Selects: false,
		},
		{
			Endpoint: &MoveEndpointInModule{
				relSubject: mustParseModuleInstanceStr(`module.bar.module.baz["key"]`),
			},
			Addr:    mustParseModuleInstanceStr(`module.bar.module.baz["nope"]`),
			Selects: false,
		},
		{
			Endpoint: &MoveEndpointInModule{
				relSubject: mustParseAbsResourceInstanceStr(`module.nope.module.baz["key"].resource.name`).ContainingResource(),
			},
			Addr:    mustParseModuleInstanceStr(`module.bar.module.baz["key"]`),
			Selects: false,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("[%02d]%s.SelectsModule(%s)", i, test.Endpoint, test.Addr),
			func(t *testing.T) {
				if test.Endpoint.SelectsModule(test.Addr) != test.Selects {
					t.Errorf("expected %s SelectsModule %s == %t", test.Endpoint, test.Addr, test.Selects)
				}
			},
		)
	}
}

func TestSelectsResource(t *testing.T) {
	matchingResource := Resource{
		Mode: ManagedResourceMode,
		Type: "foo",
		Name: "matching",
	}
	unmatchingResource := Resource{
		Mode: ManagedResourceMode,
		Type: "foo",
		Name: "unmatching",
	}
	childMod := Module{
		"child",
	}
	childModMatchingInst := ModuleInstance{
		ModuleInstanceStep{Name: "child", InstanceKey: StringKey("matching")},
	}
	childModUnmatchingInst := ModuleInstance{
		ModuleInstanceStep{Name: "child", InstanceKey: StringKey("unmatching")},
	}

	tests := []struct {
		Endpoint *MoveEndpointInModule
		Addr     AbsResource
		Selects  bool
	}{
		{
			Endpoint: &MoveEndpointInModule{
				relSubject: matchingResource.Absolute(nil),
			},
			Addr:    matchingResource.Absolute(nil),
			Selects: true, // exact match
		},
		{
			Endpoint: &MoveEndpointInModule{
				relSubject: unmatchingResource.Absolute(nil),
			},
			Addr:    matchingResource.Absolute(nil),
			Selects: false, // wrong resource name
		},
		{
			Endpoint: &MoveEndpointInModule{
				relSubject: unmatchingResource.Instance(IntKey(1)).Absolute(nil),
			},
			Addr:    matchingResource.Absolute(nil),
			Selects: false, // wrong resource name
		},
		{
			Endpoint: &MoveEndpointInModule{
				relSubject: matchingResource.Instance(NoKey).Absolute(nil),
			},
			Addr:    matchingResource.Absolute(nil),
			Selects: true, // matches one instance
		},
		{
			Endpoint: &MoveEndpointInModule{
				relSubject: matchingResource.Instance(IntKey(0)).Absolute(nil),
			},
			Addr:    matchingResource.Absolute(nil),
			Selects: true, // matches one instance
		},
		{
			Endpoint: &MoveEndpointInModule{
				relSubject: matchingResource.Instance(StringKey("a")).Absolute(nil),
			},
			Addr:    matchingResource.Absolute(nil),
			Selects: true, // matches one instance
		},
		{
			Endpoint: &MoveEndpointInModule{
				module:     childMod,
				relSubject: matchingResource.Absolute(nil),
			},
			Addr:    matchingResource.Absolute(childModMatchingInst),
			Selects: true, // in one of the instances of the module where the statement was written
		},
		{
			Endpoint: &MoveEndpointInModule{
				relSubject: matchingResource.Absolute(childModMatchingInst),
			},
			Addr:    matchingResource.Absolute(childModMatchingInst),
			Selects: true, // exact match
		},
		{
			Endpoint: &MoveEndpointInModule{
				relSubject: matchingResource.Instance(IntKey(2)).Absolute(childModMatchingInst),
			},
			Addr:    matchingResource.Absolute(childModMatchingInst),
			Selects: true, // matches one instance
		},
		{
			Endpoint: &MoveEndpointInModule{
				relSubject: matchingResource.Absolute(childModMatchingInst),
			},
			Addr:    matchingResource.Absolute(childModUnmatchingInst),
			Selects: false, // the containing module instance doesn't match
		},
		{
			Endpoint: &MoveEndpointInModule{
				relSubject: AbsModuleCall{
					Module: mustParseModuleInstanceStr("module.foo[2]"),
					Call:   ModuleCall{Name: "bar"},
				},
			},
			Addr:    matchingResource.Absolute(mustParseModuleInstanceStr("module.foo[2]")),
			Selects: false, // a module call can't match a resource
		},
		{
			Endpoint: &MoveEndpointInModule{
				relSubject: mustParseModuleInstanceStr("module.foo[2]"),
			},
			Addr:    matchingResource.Absolute(mustParseModuleInstanceStr("module.foo[2]")),
			Selects: false, // a module instance can't match a resource
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("[%02d]%s SelectsResource(%s)", i, test.Endpoint, test.Addr),
			func(t *testing.T) {
				if got, want := test.Endpoint.SelectsResource(test.Addr), test.Selects; got != want {
					t.Errorf("wrong result\nReceiver: %s\nArgument: %s\ngot:  %t\nwant: %t", test.Endpoint, test.Addr, got, want)
				}
			},
		)
	}
}

func mustParseAbsResourceInstanceStr(s string) AbsResourceInstance {
	r, diags := ParseAbsResourceInstanceStr(s)
	if diags.HasErrors() {
		panic(diags.ErrWithWarnings().Error())
	}
	return r
}
