package terraform

import (
	"flag"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/hashicorp/terraform/internal/registry"
	"github.com/hashicorp/terraform/internal/states"

	_ "github.com/hashicorp/terraform/internal/logging"
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

	// Test modules usually do not refer to remote sources, and for local
	// sources only this ultimately just records all of the module paths
	// in a JSON file so that we can load them below.
	inst := initwd.NewModuleInstaller(loader.ModulesDir(), registry.NewClient(nil, nil))
	_, instDiags := inst.InstallModules(dir, true, initwd.ModuleInstallHooksImpl{})
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

	cfgPath, err := ioutil.TempDir("", "tf-test")
	if err != nil {
		t.Errorf("Error creating temporary directory for config: %s", err)
	}
	defer os.RemoveAll(cfgPath)

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

	// Test modules usually do not refer to remote sources, and for local
	// sources only this ultimately just records all of the module paths
	// in a JSON file so that we can load them below.
	inst := initwd.NewModuleInstaller(loader.ModulesDir(), registry.NewClient(nil, nil))
	_, instDiags := inst.InstallModules(cfgPath, true, initwd.ModuleInstallHooksImpl{})
	if instDiags.HasErrors() {
		t.Fatal(instDiags.Err())
	}

	// Since module installer has modified the module manifest on disk, we need
	// to refresh the cache of it in the loader.
	if err := loader.RefreshModules(); err != nil {
		t.Fatalf("failed to refresh modules after installation: %s", err)
	}

	config, diags := loader.LoadConfig(cfgPath)
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
	return func() (providers.Interface, error) {
		return rp, nil
	}
}

func testProvisionerFuncFixed(rp *MockProvisioner) provisioners.Factory {
	return func() (provisioners.Interface, error) {
		// make sure this provisioner has has not been closed
		rp.CloseCalled = false
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

const testTerraformInputProviderOnlyStr = `
aws_instance.foo:
  ID = 
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = us-west-2
  type = 
`

const testTerraformApplyStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  num = 2
  type = aws_instance
`

const testTerraformApplyDataBasicStr = `
data.null_data_source.testing:
  ID = yo
  provider = provider["registry.terraform.io/hashicorp/null"]
`

const testTerraformApplyRefCountStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = 3
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo.0:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
aws_instance.foo.1:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
aws_instance.foo.2:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
`

const testTerraformApplyProviderAliasStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"].bar
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  num = 2
  type = aws_instance
`

const testTerraformApplyProviderAliasConfigStr = `
another_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/another"].two
  type = another_instance
another_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/another"]
  type = another_instance
`

const testTerraformApplyEmptyModuleStr = `
<no state>
Outputs:

end = XXXX
`

const testTerraformApplyDependsCreateBeforeStr = `
aws_instance.lb:
  ID = baz
  provider = provider["registry.terraform.io/hashicorp/aws"]
  instance = foo
  type = aws_instance

  Dependencies:
    aws_instance.web
aws_instance.web:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  require_new = ami-new
  type = aws_instance
`

const testTerraformApplyCreateBeforeStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  require_new = xyz
  type = aws_instance
`

const testTerraformApplyCreateBeforeUpdateStr = `
aws_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = baz
  type = aws_instance
`

const testTerraformApplyCancelStr = `
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
  value = 2
`

const testTerraformApplyComputeStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = computed_value
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  compute = value
  compute_value = 1
  num = 2
  type = aws_instance
  value = computed_value
`

const testTerraformApplyCountDecStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.foo.0:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
aws_instance.foo.1:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
`

const testTerraformApplyCountDecToOneStr = `
aws_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
`

const testTerraformApplyCountDecToOneCorruptedStr = `
aws_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
`

const testTerraformApplyCountDecToOneCorruptedPlanStr = `
DIFF:

DESTROY: aws_instance.foo[0]
  id:   "baz" => ""
  type: "aws_instance" => ""



STATE:

aws_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
aws_instance.foo.0:
  ID = baz
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
`

const testTerraformApplyCountVariableStr = `
aws_instance.foo.0:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
aws_instance.foo.1:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
`

const testTerraformApplyCountVariableRefStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = 2
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo.0:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
aws_instance.foo.1:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
`
const testTerraformApplyForEachVariableStr = `
aws_instance.foo["b15c6d616d6143248c575900dff57325eb1de498"]:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
aws_instance.foo["c3de47d34b0a9f13918dd705c141d579dd6555fd"]:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
aws_instance.foo["e30a7edcc42a846684f2a4eea5f3cd261d33c46d"]:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = foo
  type = aws_instance
aws_instance.one["a"]:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
aws_instance.one["b"]:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
aws_instance.two["a"]:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance

  Dependencies:
    aws_instance.one
aws_instance.two["b"]:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance

  Dependencies:
    aws_instance.one`
const testTerraformApplyMinimalStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
`

const testTerraformApplyModuleStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  num = 2
  type = aws_instance

module.child:
  aws_instance.baz:
    ID = foo
    provider = provider["registry.terraform.io/hashicorp/aws"]
    foo = bar
    type = aws_instance
`

const testTerraformApplyModuleBoolStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = true
  type = aws_instance
`

const testTerraformApplyModuleDestroyOrderStr = `
<no state>
`

const testTerraformApplyMultiProviderStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
do_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/do"]
  num = 2
  type = do_instance
`

const testTerraformApplyModuleOnlyProviderStr = `
<no state>
module.child:
  aws_instance.foo:
    ID = foo
    provider = provider["registry.terraform.io/hashicorp/aws"]
    type = aws_instance
  test_instance.foo:
    ID = foo
    provider = provider["registry.terraform.io/hashicorp/test"]
    type = test_instance
`

const testTerraformApplyModuleProviderAliasStr = `
<no state>
module.child:
  aws_instance.foo:
    ID = foo
    provider = module.child.provider["registry.terraform.io/hashicorp/aws"].eu
    type = aws_instance
`

const testTerraformApplyModuleVarRefExistingStr = `
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance

module.child:
  aws_instance.foo:
    ID = foo
    provider = provider["registry.terraform.io/hashicorp/aws"]
    type = aws_instance
    value = bar

    Dependencies:
      aws_instance.foo
`

const testTerraformApplyOutputOrphanStr = `
<no state>
Outputs:

foo = bar
`

const testTerraformApplyOutputOrphanModuleStr = `
<no state>
`

const testTerraformApplyProvisionerStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  compute = value
  compute_value = 1
  num = 2
  type = aws_instance
  value = computed_value
`

const testTerraformApplyProvisionerModuleStr = `
<no state>
module.child:
  aws_instance.bar:
    ID = foo
    provider = provider["registry.terraform.io/hashicorp/aws"]
    type = aws_instance
`

const testTerraformApplyProvisionerFailStr = `
aws_instance.bar: (tainted)
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  num = 2
  type = aws_instance
`

const testTerraformApplyProvisionerFailCreateStr = `
aws_instance.bar: (tainted)
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
`

const testTerraformApplyProvisionerFailCreateNoIdStr = `
<no state>
`

const testTerraformApplyProvisionerFailCreateBeforeDestroyStr = `
aws_instance.bar: (tainted) (1 deposed)
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  require_new = xyz
  type = aws_instance
  Deposed ID 1 = bar
`

const testTerraformApplyProvisionerResourceRefStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  num = 2
  type = aws_instance
`

const testTerraformApplyProvisionerSelfRefStr = `
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
`

const testTerraformApplyProvisionerMultiSelfRefStr = `
aws_instance.foo.0:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = number 0
  type = aws_instance
aws_instance.foo.1:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = number 1
  type = aws_instance
aws_instance.foo.2:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = number 2
  type = aws_instance
`

const testTerraformApplyProvisionerMultiSelfRefSingleStr = `
aws_instance.foo.0:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = number 0
  type = aws_instance
aws_instance.foo.1:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = number 1
  type = aws_instance
aws_instance.foo.2:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = number 2
  type = aws_instance
`

const testTerraformApplyProvisionerDiffStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
`

const testTerraformApplyProvisionerSensitiveStr = `
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
`

const testTerraformApplyDestroyStr = `
<no state>
`

const testTerraformApplyErrorStr = `
aws_instance.bar: (tainted)
  ID = 
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = 2

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
  value = 2
`

const testTerraformApplyErrorCreateBeforeDestroyStr = `
aws_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/aws"]
  require_new = abc
  type = aws_instance
`

const testTerraformApplyErrorDestroyCreateBeforeDestroyStr = `
aws_instance.bar: (1 deposed)
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  require_new = xyz
  type = aws_instance
  Deposed ID 1 = bar
`

const testTerraformApplyErrorPartialStr = `
aws_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  type = aws_instance
  value = 2
`

const testTerraformApplyResourceDependsOnModuleStr = `
aws_instance.a:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  ami = parent
  type = aws_instance

  Dependencies:
    module.child.aws_instance.child

module.child:
  aws_instance.child:
    ID = foo
    provider = provider["registry.terraform.io/hashicorp/aws"]
    ami = child
    type = aws_instance
`

const testTerraformApplyResourceDependsOnModuleDeepStr = `
aws_instance.a:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  ami = parent
  type = aws_instance

  Dependencies:
    module.child.module.grandchild.aws_instance.c

module.child.grandchild:
  aws_instance.c:
    ID = foo
    provider = provider["registry.terraform.io/hashicorp/aws"]
    ami = grandchild
    type = aws_instance
`

const testTerraformApplyResourceDependsOnModuleInModuleStr = `
<no state>
module.child:
  aws_instance.b:
    ID = foo
    provider = provider["registry.terraform.io/hashicorp/aws"]
    ami = child
    type = aws_instance

    Dependencies:
      module.child.module.grandchild.aws_instance.c
module.child.grandchild:
  aws_instance.c:
    ID = foo
    provider = provider["registry.terraform.io/hashicorp/aws"]
    ami = grandchild
    type = aws_instance
`

const testTerraformApplyTaintStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  num = 2
  type = aws_instance
`

const testTerraformApplyTaintDepStr = `
aws_instance.bar:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = foo
  num = 2
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  num = 2
  type = aws_instance
`

const testTerraformApplyTaintDepRequireNewStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = foo
  require_new = yes
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  num = 2
  type = aws_instance
`

const testTerraformApplyOutputStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  num = 2
  type = aws_instance

Outputs:

foo_num = 2
`

const testTerraformApplyOutputAddStr = `
aws_instance.test.0:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = foo0
  type = aws_instance
aws_instance.test.1:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = foo1
  type = aws_instance

Outputs:

firstOutput = foo0
secondOutput = foo1
`

const testTerraformApplyOutputListStr = `
aws_instance.bar.0:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.bar.1:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.bar.2:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  num = 2
  type = aws_instance

Outputs:

foo_num = [bar,bar,bar]
`

const testTerraformApplyOutputMultiStr = `
aws_instance.bar.0:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.bar.1:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.bar.2:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  num = 2
  type = aws_instance

Outputs:

foo_num = bar,bar,bar
`

const testTerraformApplyOutputMultiIndexStr = `
aws_instance.bar.0:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.bar.1:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.bar.2:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  num = 2
  type = aws_instance

Outputs:

foo_num = bar
`

const testTerraformApplyUnknownAttrStr = `
aws_instance.foo: (tainted)
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  num = 2
  type = aws_instance
`

const testTerraformApplyVarsStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  bar = override
  baz = override
  foo = us-east-1
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  bar = baz
  list.# = 2
  list.0 = Hello
  list.1 = World
  map.Baz = Foo
  map.Foo = Bar
  map.Hello = World
  num = 2
`

const testTerraformApplyVarsEnvStr = `
aws_instance.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  list.# = 2
  list.0 = Hello
  list.1 = World
  map.Baz = Foo
  map.Foo = Bar
  map.Hello = World
  string = baz
  type = aws_instance
`

const testTerraformRefreshDataRefDataStr = `
data.null_data_source.bar:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/null"]
  bar = yes
data.null_data_source.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/null"]
  foo = yes
`
