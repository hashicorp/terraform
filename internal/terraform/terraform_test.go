// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mnptu

import (
	"context"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/mnptu/internal/addrs"
	"github.com/hashicorp/mnptu/internal/configs"
	"github.com/hashicorp/mnptu/internal/configs/configload"
	"github.com/hashicorp/mnptu/internal/initwd"
	"github.com/hashicorp/mnptu/internal/plans"
	"github.com/hashicorp/mnptu/internal/providers"
	"github.com/hashicorp/mnptu/internal/provisioners"
	"github.com/hashicorp/mnptu/internal/registry"
	"github.com/hashicorp/mnptu/internal/states"

	_ "github.com/hashicorp/mnptu/internal/logging"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./testdata"

func TestMain(m *testing.M) {
	flag.Parse()

	// We have fmt.Stringer implementations on lots of objects that hide
	// details that we very often want to see in tests, so we just disable
	// spew's use of String methods globally on the assumption that spew
	// usage implies an intent to see the raw values and ignore any
	// abstractions.
	spew.Config.DisableMethods = true

	os.Exit(m.Run())
}

func testModule(t *testing.T, name string) *configs.Config {
	t.Helper()
	c, _ := testModuleWithSnapshot(t, name)
	return c
}

func testModuleWithSnapshot(t *testing.T, name string) (*configs.Config, *configload.Snapshot) {
	t.Helper()

	dir := filepath.Join(fixtureDir, name)
	// FIXME: We're not dealing with the cleanup function here because
	// this testModule function is used all over and so we don't want to
	// change its interface at this late stage.
	loader, _ := configload.NewLoaderForTests(t)

	// We need to be able to exercise experimental features in our integration tests.
	loader.AllowLanguageExperiments(true)

	// Test modules usually do not refer to remote sources, and for local
	// sources only this ultimately just records all of the module paths
	// in a JSON file so that we can load them below.
	inst := initwd.NewModuleInstaller(loader.ModulesDir(), loader, registry.NewClient(nil, nil))
	_, instDiags := inst.InstallModules(context.Background(), dir, "tests", true, false, initwd.ModuleInstallHooksImpl{})
	if instDiags.HasErrors() {
		t.Fatal(instDiags.Err())
	}

	// Since module installer has modified the module manifest on disk, we need
	// to refresh the cache of it in the loader.
	if err := loader.RefreshModules(); err != nil {
		t.Fatalf("failed to refresh modules after installation: %s", err)
	}

	config, snap, diags := loader.LoadConfigWithSnapshot(dir)
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	return config, snap
}

// testModuleInline takes a map of path -> config strings and yields a config
// structure with those files loaded from disk
func testModuleInline(t *testing.T, sources map[string]string) *configs.Config {
	t.Helper()

	cfgPath := t.TempDir()

	for path, configStr := range sources {
		dir := filepath.Dir(path)
		if dir != "." {
			err := os.MkdirAll(filepath.Join(cfgPath, dir), os.FileMode(0777))
			if err != nil {
				t.Fatalf("Error creating subdir: %s", err)
			}
		}
		// Write the configuration
		cfgF, err := os.Create(filepath.Join(cfgPath, path))
		if err != nil {
			t.Fatalf("Error creating temporary file for config: %s", err)
		}

		_, err = io.Copy(cfgF, strings.NewReader(configStr))
		cfgF.Close()
		if err != nil {
			t.Fatalf("Error creating temporary file for config: %s", err)
		}
	}

	loader, cleanup := configload.NewLoaderForTests(t)
	defer cleanup()

	// We need to be able to exercise experimental features in our integration tests.
	loader.AllowLanguageExperiments(true)

	// Test modules usually do not refer to remote sources, and for local
	// sources only this ultimately just records all of the module paths
	// in a JSON file so that we can load them below.
	inst := initwd.NewModuleInstaller(loader.ModulesDir(), loader, registry.NewClient(nil, nil))
	_, instDiags := inst.InstallModules(context.Background(), cfgPath, "tests", true, false, initwd.ModuleInstallHooksImpl{})
	if instDiags.HasErrors() {
		t.Fatal(instDiags.Err())
	}

	// Since module installer has modified the module manifest on disk, we need
	// to refresh the cache of it in the loader.
	if err := loader.RefreshModules(); err != nil {
		t.Fatalf("failed to refresh modules after installation: %s", err)
	}

	config, diags := loader.LoadConfigWithTests(cfgPath, "tests")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	return config
}

// testSetResourceInstanceCurrent is a helper function for tests that sets a Current,
// Ready resource instance for the given module.
func testSetResourceInstanceCurrent(module *states.Module, resource, attrsJson, provider string) {
	module.SetResourceInstanceCurrent(
		mustResourceInstanceAddr(resource).Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(attrsJson),
		},
		mustProviderConfig(provider),
	)
}

// testSetResourceInstanceTainted is a helper function for tests that sets a Current,
// Tainted resource instance for the given module.
func testSetResourceInstanceTainted(module *states.Module, resource, attrsJson, provider string) {
	module.SetResourceInstanceCurrent(
		mustResourceInstanceAddr(resource).Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectTainted,
			AttrsJSON: []byte(attrsJson),
		},
		mustProviderConfig(provider),
	)
}

func testProviderFuncFixed(rp providers.Interface) providers.Factory {
	if p, ok := rp.(*MockProvider); ok {
		// make sure none of the methods were "called" on this new instance
		p.GetProviderSchemaCalled = false
		p.ValidateProviderConfigCalled = false
		p.ValidateResourceConfigCalled = false
		p.ValidateDataResourceConfigCalled = false
		p.UpgradeResourceStateCalled = false
		p.ConfigureProviderCalled = false
		p.StopCalled = false
		p.ReadResourceCalled = false
		p.PlanResourceChangeCalled = false
		p.ApplyResourceChangeCalled = false
		p.ImportResourceStateCalled = false
		p.ReadDataSourceCalled = false
		p.CloseCalled = false
	}

	return func() (providers.Interface, error) {
		return rp, nil
	}
}

func testProvisionerFuncFixed(rp *MockProvisioner) provisioners.Factory {
	// make sure this provisioner has has not been closed
	rp.CloseCalled = false

	return func() (provisioners.Interface, error) {
		return rp, nil
	}
}

func mustResourceInstanceAddr(s string) addrs.AbsResourceInstance {
	addr, diags := addrs.ParseAbsResourceInstanceStr(s)
	if diags.HasErrors() {
		panic(diags.Err())
	}
	return addr
}

func mustConfigResourceAddr(s string) addrs.ConfigResource {
	addr, diags := addrs.ParseAbsResourceStr(s)
	if diags.HasErrors() {
		panic(diags.Err())
	}
	return addr.Config()
}

func mustAbsResourceAddr(s string) addrs.AbsResource {
	addr, diags := addrs.ParseAbsResourceStr(s)
	if diags.HasErrors() {
		panic(diags.Err())
	}
	return addr
}

func mustProviderConfig(s string) addrs.AbsProviderConfig {
	p, diags := addrs.ParseAbsProviderConfigStr(s)
	if diags.HasErrors() {
		panic(diags.Err())
	}
	return p
}

func mustReference(s string) *addrs.Reference {
	p, diags := addrs.ParseRefStr(s)
	if diags.HasErrors() {
		panic(diags.Err())
	}
	return p
}

// HookRecordApplyOrder is a test hook that records the order of applies
// by recording the PreApply event.
type HookRecordApplyOrder struct {
	NilHook

	Active bool

	IDs    []string
	States []cty.Value
	Diffs  []*plans.Change

	l sync.Mutex
}

func (h *HookRecordApplyOrder) PreApply(addr addrs.AbsResourceInstance, gen states.Generation, action plans.Action, priorState, plannedNewState cty.Value) (HookAction, error) {
	if plannedNewState.RawEquals(priorState) {
		return HookActionContinue, nil
	}

	if h.Active {
		h.l.Lock()
		defer h.l.Unlock()

		h.IDs = append(h.IDs, addr.String())
		h.Diffs = append(h.Diffs, &plans.Change{
			Action: action,
			Before: priorState,
			After:  plannedNewState,
		})
		h.States = append(h.States, priorState)
	}

	return HookActionContinue, nil
}

// Below are all the constant strings that are the expected output for
// various tests.

const testmnptuInputProviderOnlyStr = `
aws_instance.foo:
  ID = 
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = us-west-2
  type = 
`

const testmnptuApplyStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  num = 2
  type = aws_instance
`

const testmnptuApplyDataBasicStr = `
data.null_data_source.testing:
  ID = yo
  provider = provider["registry.mnptu.io/hashicorp/null"]
`

const testmnptuApplyRefCountStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = 3
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo.0:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance
aws_instance.foo.1:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance
aws_instance.foo.2:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance
`

const testmnptuApplyProviderAliasStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"].bar
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  num = 2
  type = aws_instance
`

const testmnptuApplyProviderAliasConfigStr = `
another_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/another"].two
  type = another_instance
another_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/another"]
  type = another_instance
`

const testmnptuApplyEmptyModuleStr = `
<no state>
Outputs:

end = XXXX
`

const testmnptuApplyDependsCreateBeforeStr = `
aws_instance.lb:
  ID = baz
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  instance = foo
  type = aws_instance

  Dependencies:
    aws_instance.web
aws_instance.web:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  require_new = ami-new
  type = aws_instance
`

const testmnptuApplyCreateBeforeStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  require_new = xyz
  type = aws_instance
`

const testmnptuApplyCreateBeforeUpdateStr = `
aws_instance.bar:
  ID = bar
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = baz
  type = aws_instance
`

const testmnptuApplyCancelStr = `
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance
  value = 2
`

const testmnptuApplyComputeStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = computed_value
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  compute = value
  compute_value = 1
  num = 2
  type = aws_instance
  value = computed_value
`

const testmnptuApplyCountDecStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.foo.0:
  ID = bar
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
aws_instance.foo.1:
  ID = bar
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
`

const testmnptuApplyCountDecToOneStr = `
aws_instance.foo:
  ID = bar
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
`

const testmnptuApplyCountDecToOneCorruptedStr = `
aws_instance.foo:
  ID = bar
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
`

const testmnptuApplyCountDecToOneCorruptedPlanStr = `
DIFF:

DESTROY: aws_instance.foo[0]
  id:   "baz" => ""
  type: "aws_instance" => ""



STATE:

aws_instance.foo:
  ID = bar
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
aws_instance.foo.0:
  ID = baz
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance
`

const testmnptuApplyCountVariableStr = `
aws_instance.foo.0:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
aws_instance.foo.1:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
`

const testmnptuApplyCountVariableRefStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = 2
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo.0:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance
aws_instance.foo.1:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance
`
const testmnptuApplyForEachVariableStr = `
aws_instance.foo["b15c6d616d6143248c575900dff57325eb1de498"]:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
aws_instance.foo["c3de47d34b0a9f13918dd705c141d579dd6555fd"]:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
aws_instance.foo["e30a7edcc42a846684f2a4eea5f3cd261d33c46d"]:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
aws_instance.one["a"]:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance
aws_instance.one["b"]:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance
aws_instance.two["a"]:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance

  Dependencies:
    aws_instance.one
aws_instance.two["b"]:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance

  Dependencies:
    aws_instance.one`
const testmnptuApplyMinimalStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance
`

const testmnptuApplyModuleStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  num = 2
  type = aws_instance

module.child:
  aws_instance.baz:
    ID = foo
    provider = provider["registry.mnptu.io/hashicorp/aws"]
    foo = bar
    type = aws_instance
`

const testmnptuApplyModuleBoolStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = true
  type = aws_instance
`

const testmnptuApplyModuleDestroyOrderStr = `
<no state>
`

const testmnptuApplyMultiProviderStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
do_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/do"]
  num = 2
  type = do_instance
`

const testmnptuApplyModuleOnlyProviderStr = `
<no state>
module.child:
  aws_instance.foo:
    ID = foo
    provider = provider["registry.mnptu.io/hashicorp/aws"]
    type = aws_instance
  test_instance.foo:
    ID = foo
    provider = provider["registry.mnptu.io/hashicorp/test"]
    type = test_instance
`

const testmnptuApplyModuleProviderAliasStr = `
<no state>
module.child:
  aws_instance.foo:
    ID = foo
    provider = module.child.provider["registry.mnptu.io/hashicorp/aws"].eu
    type = aws_instance
`

const testmnptuApplyModuleVarRefExistingStr = `
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = bar
  type = aws_instance

module.child:
  aws_instance.foo:
    ID = foo
    provider = provider["registry.mnptu.io/hashicorp/aws"]
    type = aws_instance
    value = bar

    Dependencies:
      aws_instance.foo
`

const testmnptuApplyOutputOrphanStr = `
<no state>
Outputs:

foo = bar
`

const testmnptuApplyOutputOrphanModuleStr = `
<no state>
`

const testmnptuApplyProvisionerStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  compute = value
  compute_value = 1
  num = 2
  type = aws_instance
  value = computed_value
`

const testmnptuApplyProvisionerModuleStr = `
<no state>
module.child:
  aws_instance.bar:
    ID = foo
    provider = provider["registry.mnptu.io/hashicorp/aws"]
    type = aws_instance
`

const testmnptuApplyProvisionerFailStr = `
aws_instance.bar: (tainted)
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  num = 2
  type = aws_instance
`

const testmnptuApplyProvisionerFailCreateStr = `
aws_instance.bar: (tainted)
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance
`

const testmnptuApplyProvisionerFailCreateNoIdStr = `
<no state>
`

const testmnptuApplyProvisionerFailCreateBeforeDestroyStr = `
aws_instance.bar: (tainted) (1 deposed)
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  require_new = xyz
  type = aws_instance
  Deposed ID 1 = bar
`

const testmnptuApplyProvisionerResourceRefStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  num = 2
  type = aws_instance
`

const testmnptuApplyProvisionerSelfRefStr = `
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
`

const testmnptuApplyProvisionerMultiSelfRefStr = `
aws_instance.foo.0:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = number 0
  type = aws_instance
aws_instance.foo.1:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = number 1
  type = aws_instance
aws_instance.foo.2:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = number 2
  type = aws_instance
`

const testmnptuApplyProvisionerMultiSelfRefSingleStr = `
aws_instance.foo.0:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = number 0
  type = aws_instance
aws_instance.foo.1:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = number 1
  type = aws_instance
aws_instance.foo.2:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = number 2
  type = aws_instance
`

const testmnptuApplyProvisionerDiffStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
`

const testmnptuApplyProvisionerSensitiveStr = `
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance
`

const testmnptuApplyDestroyStr = `
<no state>
`

const testmnptuApplyErrorStr = `
aws_instance.bar: (tainted)
  ID = 
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = 2

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance
  value = 2
`

const testmnptuApplyErrorCreateBeforeDestroyStr = `
aws_instance.bar:
  ID = bar
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  require_new = abc
  type = aws_instance
`

const testmnptuApplyErrorDestroyCreateBeforeDestroyStr = `
aws_instance.bar: (1 deposed)
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  require_new = xyz
  type = aws_instance
  Deposed ID 1 = bar
`

const testmnptuApplyErrorPartialStr = `
aws_instance.bar:
  ID = bar
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  type = aws_instance
  value = 2
`

const testmnptuApplyResourceDependsOnModuleStr = `
aws_instance.a:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  ami = parent
  type = aws_instance

  Dependencies:
    module.child.aws_instance.child

module.child:
  aws_instance.child:
    ID = foo
    provider = provider["registry.mnptu.io/hashicorp/aws"]
    ami = child
    type = aws_instance
`

const testmnptuApplyResourceDependsOnModuleDeepStr = `
aws_instance.a:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  ami = parent
  type = aws_instance

  Dependencies:
    module.child.module.grandchild.aws_instance.c

module.child.grandchild:
  aws_instance.c:
    ID = foo
    provider = provider["registry.mnptu.io/hashicorp/aws"]
    ami = grandchild
    type = aws_instance
`

const testmnptuApplyResourceDependsOnModuleInModuleStr = `
<no state>
module.child:
  aws_instance.b:
    ID = foo
    provider = provider["registry.mnptu.io/hashicorp/aws"]
    ami = child
    type = aws_instance

    Dependencies:
      module.child.module.grandchild.aws_instance.c
module.child.grandchild:
  aws_instance.c:
    ID = foo
    provider = provider["registry.mnptu.io/hashicorp/aws"]
    ami = grandchild
    type = aws_instance
`

const testmnptuApplyTaintStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  num = 2
  type = aws_instance
`

const testmnptuApplyTaintDepStr = `
aws_instance.bar:
  ID = bar
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = foo
  num = 2
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  num = 2
  type = aws_instance
`

const testmnptuApplyTaintDepRequireNewStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = foo
  require_new = yes
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  num = 2
  type = aws_instance
`

const testmnptuApplyOutputStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  num = 2
  type = aws_instance

Outputs:

foo_num = 2
`

const testmnptuApplyOutputAddStr = `
aws_instance.test.0:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = foo0
  type = aws_instance
aws_instance.test.1:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = foo1
  type = aws_instance

Outputs:

firstOutput = foo0
secondOutput = foo1
`

const testmnptuApplyOutputListStr = `
aws_instance.bar.0:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.bar.1:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.bar.2:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  num = 2
  type = aws_instance

Outputs:

foo_num = [bar,bar,bar]
`

const testmnptuApplyOutputMultiStr = `
aws_instance.bar.0:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.bar.1:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.bar.2:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  num = 2
  type = aws_instance

Outputs:

foo_num = bar,bar,bar
`

const testmnptuApplyOutputMultiIndexStr = `
aws_instance.bar.0:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.bar.1:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.bar.2:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  num = 2
  type = aws_instance

Outputs:

foo_num = bar
`

const testmnptuApplyUnknownAttrStr = `
aws_instance.foo: (tainted)
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  num = 2
  type = aws_instance
`

const testmnptuApplyVarsStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  bar = override
  baz = override
  foo = us-east-1
aws_instance.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  bar = baz
  list.# = 2
  list.0 = Hello
  list.1 = World
  map.Baz = Foo
  map.Foo = Bar
  map.Hello = World
  num = 2
`

const testmnptuApplyVarsEnvStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/aws"]
  list.# = 2
  list.0 = Hello
  list.1 = World
  map.Baz = Foo
  map.Foo = Bar
  map.Hello = World
  string = baz
  type = aws_instance
`

const testmnptuRefreshDataRefDataStr = `
data.null_data_source.bar:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/null"]
  bar = yes
data.null_data_source.foo:
  ID = foo
  provider = provider["registry.mnptu.io/hashicorp/null"]
  foo = yes
`
