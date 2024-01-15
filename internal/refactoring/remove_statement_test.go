// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package refactoring

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestFindRemoveStatements(t *testing.T) {
	// LAZY: We don't need the expanded instances from loadRefactoringFixture
	// but reuse the helper function anyway.
	rootCfg, _ := loadRefactoringFixture(t, "testdata/remove-statements")

	configResourceBasic := addrs.ConfigResource{
		Module: addrs.RootModule,
		Resource: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_resource",
			Name: "foo",
		},
	}

	configResourceWithModule := addrs.ConfigResource{
		Module: addrs.Module{"gone"},
		Resource: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_resource",
			Name: "bar",
		},
	}

	configModuleBasic := addrs.Module{"gone", "gonechild"}

	configResourceOverridden := addrs.ConfigResource{
		Module: addrs.Module{"child"},
		Resource: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_resource",
			Name: "baz",
		},
	}

	configResourceInModule := addrs.ConfigResource{
		Module: addrs.Module{"child"},
		Resource: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_resource",
			Name: "boo",
		},
	}

	configModuleInModule := addrs.Module{"child", "grandchild"}

	want := addrs.MakeMap[addrs.ConfigMoveable, RemoveStatement](
		addrs.MakeMapElem[addrs.ConfigMoveable, RemoveStatement](configResourceBasic, RemoveStatement{
			From:    configResourceBasic,
			Destroy: false,
			DeclRange: tfdiags.SourceRangeFromHCL(hcl.Range{
				Filename: "testdata/remove-statements/main.tf",
				Start:    hcl.Pos{Line: 2, Column: 1, Byte: 27},
				End:      hcl.Pos{Line: 2, Column: 8, Byte: 34},
			}),
		}),
		addrs.MakeMapElem[addrs.ConfigMoveable, RemoveStatement](configResourceWithModule, RemoveStatement{
			From:    configResourceWithModule,
			Destroy: false,
			DeclRange: tfdiags.SourceRangeFromHCL(hcl.Range{
				Filename: "testdata/remove-statements/main.tf",
				Start:    hcl.Pos{Line: 10, Column: 1, Byte: 138},
				End:      hcl.Pos{Line: 10, Column: 8, Byte: 145},
			}),
		}),
		addrs.MakeMapElem[addrs.ConfigMoveable, RemoveStatement](configModuleBasic, RemoveStatement{
			From:    configModuleBasic,
			Destroy: false,
			DeclRange: tfdiags.SourceRangeFromHCL(hcl.Range{
				Filename: "testdata/remove-statements/main.tf",
				Start:    hcl.Pos{Line: 18, Column: 1, Byte: 253},
				End:      hcl.Pos{Line: 18, Column: 8, Byte: 260},
			}),
		}),
		addrs.MakeMapElem[addrs.ConfigMoveable, RemoveStatement](configResourceOverridden, RemoveStatement{
			From:    configResourceOverridden,
			Destroy: true,
			DeclRange: tfdiags.SourceRangeFromHCL(hcl.Range{
				Filename: "testdata/remove-statements/main.tf", // the statement in the parent module takes precedence
				Start:    hcl.Pos{Line: 30, Column: 1, Byte: 428},
				End:      hcl.Pos{Line: 30, Column: 8, Byte: 435},
			}),
		}),
		addrs.MakeMapElem[addrs.ConfigMoveable, RemoveStatement](configResourceInModule, RemoveStatement{
			From:    configResourceInModule,
			Destroy: true,
			DeclRange: tfdiags.SourceRangeFromHCL(hcl.Range{
				Filename: "testdata/remove-statements/child/main.tf",
				Start:    hcl.Pos{Line: 10, Column: 1, Byte: 141},
				End:      hcl.Pos{Line: 10, Column: 8, Byte: 148},
			}),
		}),
		addrs.MakeMapElem[addrs.ConfigMoveable, RemoveStatement](configModuleInModule, RemoveStatement{
			From:    configModuleInModule,
			Destroy: false,
			DeclRange: tfdiags.SourceRangeFromHCL(hcl.Range{
				Filename: "testdata/remove-statements/child/main.tf",
				Start:    hcl.Pos{Line: 18, Column: 1, Byte: 247},
				End:      hcl.Pos{Line: 18, Column: 8, Byte: 254},
			}),
		}),
	)

	got, diags := FindRemoveStatements(rootCfg)
	if diags.HasErrors() {
		t.Fatal(diags.Err().Error())
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}
