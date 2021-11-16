package refactoring

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/registry"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty/gocty"
)

func TestValidateMoves(t *testing.T) {
	rootCfg, instances := loadRefactoringFixture(t, "testdata/move-validate-zoo")

	tests := map[string]struct {
		Statements []MoveStatement
		WantError  string
	}{
		"no move statements": {
			Statements: nil,
			WantError:  ``,
		},
		"some valid statements": {
			Statements: []MoveStatement{
				// This is just a grab bag of various valid cases that don't
				// generate any errors at all.
				makeTestMoveStmt(t,
					``,
					`test.nonexist1`,
					`test.target1`,
				),
				makeTestMoveStmt(t,
					`single`,
					`test.nonexist1`,
					`test.target1`,
				),
				makeTestMoveStmt(t,
					``,
					`test.nonexist2`,
					`module.nonexist.test.nonexist2`,
				),
				makeTestMoveStmt(t,
					``,
					`module.single.test.nonexist3`,
					`module.single.test.single`,
				),
				makeTestMoveStmt(t,
					``,
					`module.single.test.nonexist4`,
					`test.target2`,
				),
				makeTestMoveStmt(t,
					``,
					`test.single[0]`, // valid because test.single doesn't have "count" set
					`test.target3`,
				),
				makeTestMoveStmt(t,
					``,
					`test.zero_count[0]`, // valid because test.zero_count has count = 0
					`test.target4`,
				),
				makeTestMoveStmt(t,
					``,
					`test.zero_count[1]`, // valid because test.zero_count has count = 0
					`test.zero_count[0]`,
				),
				makeTestMoveStmt(t,
					``,
					`module.nonexist1`,
					`module.target3`,
				),
				makeTestMoveStmt(t,
					``,
					`module.nonexist1[0]`,
					`module.target4`,
				),
				makeTestMoveStmt(t,
					``,
					`module.single[0]`, // valid because module.single doesn't have "count" set
					`module.target5`,
				),
				makeTestMoveStmt(t,
					``,
					`module.for_each["nonexist1"]`,
					`module.for_each["a"]`,
				),
				makeTestMoveStmt(t,
					``,
					`module.for_each["nonexist2"]`,
					`module.nonexist.module.nonexist`,
				),
				makeTestMoveStmt(t,
					``,
					`module.for_each["nonexist3"].test.single`, // valid because module.for_each doesn't currently have a "nonexist3"
					`module.for_each["a"].test.single`,
				),
			},
			WantError: ``,
		},
		"two statements with the same endpoints": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.a`,
					`module.b`,
				),
				makeTestMoveStmt(t,
					``,
					`module.a`,
					`module.b`,
				),
			},
			WantError: ``,
		},
		"moving nowhere": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.a`,
					`module.a`,
				),
			},
			WantError: `Redundant move statement: This statement declares a move from module.a to the same address, which is the same as not declaring this move at all.`,
		},
		"cyclic chain": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.a`,
					`module.b`,
				),
				makeTestMoveStmt(t,
					``,
					`module.b`,
					`module.c`,
				),
				makeTestMoveStmt(t,
					``,
					`module.c`,
					`module.a`,
				),
			},
			WantError: `Cyclic dependency in move statements: The following chained move statements form a cycle, and so there is no final location to move objects to:
  - test:1,1: module.a[*] → module.b[*]
  - test:1,1: module.b[*] → module.c[*]
  - test:1,1: module.c[*] → module.a[*]

A chain of move statements must end with an address that doesn't appear in any other statements, and which typically also refers to an object still declared in the configuration.`,
		},
		"module.single as a call still exists in configuration": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.single`,
					`module.other`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from module.single, but that module call is still declared at testdata/move-validate-zoo/move-validate-root.tf:6,1.

Change your configuration so that this call will be declared as module.other instead.`,
		},
		"module.single as an instance still exists in configuration": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.single`,
					`module.other[0]`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from module.single, but that module instance is still declared at testdata/move-validate-zoo/move-validate-root.tf:6,1.

Change your configuration so that this instance will be declared as module.other[0] instead.`,
		},
		"module.count[0] still exists in configuration": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.count[0]`,
					`module.other`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from module.count[0], but that module instance is still declared at testdata/move-validate-zoo/move-validate-root.tf:12,12.

Change your configuration so that this instance will be declared as module.other instead.`,
		},
		`module.for_each["a"] still exists in configuration`: {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.for_each["a"]`,
					`module.other`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from module.for_each["a"], but that module instance is still declared at testdata/move-validate-zoo/move-validate-root.tf:22,14.

Change your configuration so that this instance will be declared as module.other instead.`,
		},
		"test.single as a resource still exists in configuration": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`test.single`,
					`test.other`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from test.single, but that resource is still declared at testdata/move-validate-zoo/move-validate-root.tf:27,1.

Change your configuration so that this resource will be declared as test.other instead.`,
		},
		"test.single as an instance still exists in configuration": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`test.single`,
					`test.other[0]`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from test.single, but that resource instance is still declared at testdata/move-validate-zoo/move-validate-root.tf:27,1.

Change your configuration so that this instance will be declared as test.other[0] instead.`,
		},
		"module.single.test.single as a resource still exists in configuration": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.single.test.single`,
					`test.other`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from module.single.test.single, but that resource is still declared at testdata/move-validate-zoo/child/move-validate-child.tf:6,1.

Change your configuration so that this resource will be declared as test.other instead.`,
		},
		"module.single.test.single as a resource declared in module.single still exists in configuration": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					`single`,
					`test.single`,
					`test.other`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from module.single.test.single, but that resource is still declared at testdata/move-validate-zoo/child/move-validate-child.tf:6,1.

Change your configuration so that this resource will be declared as module.single.test.other instead.`,
		},
		"module.single.test.single as an instance still exists in configuration": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.single.test.single`,
					`test.other[0]`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from module.single.test.single, but that resource instance is still declared at testdata/move-validate-zoo/child/move-validate-child.tf:6,1.

Change your configuration so that this instance will be declared as test.other[0] instead.`,
		},
		"module.count[0].test.single still exists in configuration": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.count[0].test.single`,
					`test.other`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from module.count[0].test.single, but that resource is still declared at testdata/move-validate-zoo/child/move-validate-child.tf:6,1.

Change your configuration so that this resource will be declared as test.other instead.`,
		},
		"two different moves from test.nonexist": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`test.nonexist`,
					`test.other1`,
				),
				makeTestMoveStmt(t,
					``,
					`test.nonexist`,
					`test.other2`,
				),
			},
			WantError: `Ambiguous move statements: A statement at test:1,1 declared that test.nonexist moved to test.other1, but this statement instead declares that it moved to test.other2.

Each resource can move to only one destination resource.`,
		},
		"two different moves to test.single": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`test.other1`,
					`test.single`,
				),
				makeTestMoveStmt(t,
					``,
					`test.other2`,
					`test.single`,
				),
			},
			WantError: `Ambiguous move statements: A statement at test:1,1 declared that test.other1 moved to test.single, but this statement instead declares that test.other2 moved there.

Each resource can have moved from only one source resource.`,
		},
		"two different moves to module.count[0].test.single across two modules": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`test.other1`,
					`module.count[0].test.single`,
				),
				makeTestMoveStmt(t,
					`count`,
					`test.other2`,
					`test.single`,
				),
			},
			WantError: `Ambiguous move statements: A statement at test:1,1 declared that test.other1 moved to module.count[0].test.single, but this statement instead declares that module.count[0].test.other2 moved there.

Each resource can have moved from only one source resource.`,
		},
		"move from resource in another module package": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.fake_external.test.thing`,
					`test.thing`,
				),
			},
			WantError: `Cross-package move statement: This statement declares a move from an object declared in external module package "fake-external:///". Move statements can be only within a single module package.`,
		},
		"move to resource in another module package": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`test.thing`,
					`module.fake_external.test.thing`,
				),
			},
			WantError: `Cross-package move statement: This statement declares a move to an object declared in external module package "fake-external:///". Move statements can be only within a single module package.`,
		},
		"move from module call in another module package": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.fake_external.module.a`,
					`module.b`,
				),
			},
			WantError: `Cross-package move statement: This statement declares a move from an object declared in external module package "fake-external:///". Move statements can be only within a single module package.`,
		},
		"move to module call in another module package": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.a`,
					`module.fake_external.module.b`,
				),
			},
			WantError: `Cross-package move statement: This statement declares a move to an object declared in external module package "fake-external:///". Move statements can be only within a single module package.`,
		},
		"move to a call that refers to another module package": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.nonexist`,
					`module.fake_external`,
				),
			},
			WantError: ``, // This is okay because the call itself is not considered to be inside the package it refers to
		},
		"move to instance of a call that refers to another module package": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.nonexist`,
					`module.fake_external[0]`,
				),
			},
			WantError: ``, // This is okay because the call itself is not considered to be inside the package it refers to
		},
		"resource type mismatch": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t, ``,
					`test.nonexist1`,
					`other.single`,
				),
			},
			WantError: `Resource type mismatch: This statement declares a move from test.nonexist1 to other.single, which is a resource of a different type.`,
		},
		"resource instance type mismatch": {
			Statements: []MoveStatement{
				makeTestMoveStmt(t, ``,
					`test.nonexist1[0]`,
					`other.single`,
				),
			},
			WantError: `Resource type mismatch: This statement declares a move from test.nonexist1[0] to other.single, which is a resource instance of a different type.`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotDiags := ValidateMoves(test.Statements, rootCfg, instances)

			switch {
			case test.WantError != "":
				if !gotDiags.HasErrors() {
					t.Fatalf("unexpected success\nwant error: %s", test.WantError)
				}
				if got, want := gotDiags.Err().Error(), test.WantError; got != want {
					t.Fatalf("wrong error\ngot error:  %s\nwant error: %s", got, want)
				}
			default:
				if gotDiags.HasErrors() {
					t.Fatalf("unexpected error\ngot error: %s", gotDiags.Err().Error())
				}
			}
		})
	}
}

// loadRefactoringFixture reads a configuration from the given directory and
// does some naive static processing on any count and for_each expressions
// inside, in order to get a realistic-looking instances.Set for what it
// declares without having to run a full Terraform plan.
func loadRefactoringFixture(t *testing.T, dir string) (*configs.Config, instances.Set) {
	t.Helper()

	loader, cleanup := configload.NewLoaderForTests(t)
	defer cleanup()

	inst := initwd.NewModuleInstaller(loader.ModulesDir(), registry.NewClient(nil, nil))
	_, instDiags := inst.InstallModules(context.Background(), dir, true, initwd.ModuleInstallHooksImpl{})
	if instDiags.HasErrors() {
		t.Fatal(instDiags.Err())
	}

	// Since module installer has modified the module manifest on disk, we need
	// to refresh the cache of it in the loader.
	if err := loader.RefreshModules(); err != nil {
		t.Fatalf("failed to refresh modules after installation: %s", err)
	}

	rootCfg, diags := loader.LoadConfig(dir)
	if diags.HasErrors() {
		t.Fatalf("failed to load root module: %s", diags.Error())
	}

	expander := instances.NewExpander()
	staticPopulateExpanderModule(t, rootCfg, addrs.RootModuleInstance, expander)
	return rootCfg, expander.AllInstances()
}

func staticPopulateExpanderModule(t *testing.T, rootCfg *configs.Config, moduleAddr addrs.ModuleInstance, expander *instances.Expander) {
	t.Helper()

	modCfg := rootCfg.DescendentForInstance(moduleAddr)
	if modCfg == nil {
		t.Fatalf("no configuration for %s", moduleAddr)
	}

	if len(modCfg.Path) > 0 && modCfg.Path[len(modCfg.Path)-1] == "fake_external" {
		// As a funny special case we modify the source address of this
		// module to be something that counts as a separate package,
		// so we can test rules relating to crossing package boundaries
		// even though we really just loaded the module from a local path.
		modCfg.SourceAddr = fakeExternalModuleSource
	}

	for _, call := range modCfg.Module.ModuleCalls {
		callAddr := addrs.ModuleCall{Name: call.Name}

		if call.Name == "fake_external" {
			// As a funny special case we modify the source address of this
			// module to be something that counts as a separate package,
			// so we can test rules relating to crossing package boundaries
			// even though we really just loaded the module from a local path.
			call.SourceAddr = fakeExternalModuleSource
		}

		// In order to get a valid, useful set of instances here we're going
		// to just statically evaluate the count and for_each expressions.
		// Normally it's valid to use references and functions there, but for
		// our unit tests we'll just limit it to literal values to avoid
		// bringing all of the core evaluator complexity.
		switch {
		case call.ForEach != nil:
			val, diags := call.ForEach.Value(nil)
			if diags.HasErrors() {
				t.Fatalf("invalid for_each: %s", diags.Error())
			}
			expander.SetModuleForEach(moduleAddr, callAddr, val.AsValueMap())
		case call.Count != nil:
			val, diags := call.Count.Value(nil)
			if diags.HasErrors() {
				t.Fatalf("invalid count: %s", diags.Error())
			}
			var count int
			err := gocty.FromCtyValue(val, &count)
			if err != nil {
				t.Fatalf("invalid count at %s: %s", call.Count.Range(), err)
			}
			expander.SetModuleCount(moduleAddr, callAddr, count)
		default:
			expander.SetModuleSingle(moduleAddr, callAddr)
		}

		// We need to recursively analyze the child modules too.
		calledMod := modCfg.Path.Child(call.Name)
		for _, inst := range expander.ExpandModule(calledMod) {
			staticPopulateExpanderModule(t, rootCfg, inst, expander)
		}
	}

	for _, rc := range modCfg.Module.ManagedResources {
		staticPopulateExpanderResource(t, moduleAddr, rc, expander)
	}
	for _, rc := range modCfg.Module.DataResources {
		staticPopulateExpanderResource(t, moduleAddr, rc, expander)
	}

}

func staticPopulateExpanderResource(t *testing.T, moduleAddr addrs.ModuleInstance, rCfg *configs.Resource, expander *instances.Expander) {
	t.Helper()

	addr := rCfg.Addr()
	switch {
	case rCfg.ForEach != nil:
		val, diags := rCfg.ForEach.Value(nil)
		if diags.HasErrors() {
			t.Fatalf("invalid for_each: %s", diags.Error())
		}
		expander.SetResourceForEach(moduleAddr, addr, val.AsValueMap())
	case rCfg.Count != nil:
		val, diags := rCfg.Count.Value(nil)
		if diags.HasErrors() {
			t.Fatalf("invalid count: %s", diags.Error())
		}
		var count int
		err := gocty.FromCtyValue(val, &count)
		if err != nil {
			t.Fatalf("invalid count at %s: %s", rCfg.Count.Range(), err)
		}
		expander.SetResourceCount(moduleAddr, addr, count)
	default:
		expander.SetResourceSingle(moduleAddr, addr)
	}
}

func makeTestMoveStmt(t *testing.T, moduleStr, fromStr, toStr string) MoveStatement {
	t.Helper()

	module := addrs.RootModule
	if moduleStr != "" {
		module = addrs.Module(strings.Split(moduleStr, "."))
	}

	traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(fromStr), "", hcl.InitialPos)
	if hclDiags.HasErrors() {
		t.Fatalf("invalid from address: %s", hclDiags.Error())
	}
	fromEP, diags := addrs.ParseMoveEndpoint(traversal)
	if diags.HasErrors() {
		t.Fatalf("invalid from address: %s", diags.Err().Error())
	}

	traversal, hclDiags = hclsyntax.ParseTraversalAbs([]byte(toStr), "", hcl.InitialPos)
	if hclDiags.HasErrors() {
		t.Fatalf("invalid to address: %s", hclDiags.Error())
	}
	toEP, diags := addrs.ParseMoveEndpoint(traversal)
	if diags.HasErrors() {
		t.Fatalf("invalid to address: %s", diags.Err().Error())
	}

	fromInModule, toInModule := addrs.UnifyMoveEndpoints(module, fromEP, toEP)
	if fromInModule == nil || toInModule == nil {
		t.Fatalf("incompatible move endpoints")
	}

	return MoveStatement{
		From: fromInModule,
		To:   toInModule,
		DeclRange: tfdiags.SourceRange{
			Filename: "test",
			Start:    tfdiags.SourcePos{Line: 1, Column: 1},
			End:      tfdiags.SourcePos{Line: 1, Column: 1},
		},
	}
}

var fakeExternalModuleSource = addrs.ModuleSourceRemote{
	PackageAddr: addrs.ModulePackage("fake-external:///"),
}
