package terraform

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/helper/experiment"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/provisioners"
	"github.com/hashicorp/terraform/registry"
	"github.com/hashicorp/terraform/states"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./testdata"

func TestMain(m *testing.M) {
	// We want to shadow on tests just to make sure the shadow graph works
	// in case we need it and to find any race issues.
	experiment.SetEnabled(experiment.X_shadow, true)

	experiment.Flag(flag.CommandLine)
	flag.Parse()

	if testing.Verbose() {
		// if we're verbose, use the logging requested by TF_LOG
		logging.SetOutput()
	} else {
		// otherwise silence all logs
		log.SetOutput(ioutil.Discard)
	}

	// Make sure shadow operations fail our real tests
	contextFailOnShadowError = true

	// Always DeepCopy the Diff on every Plan during a test
	contextTestDeepCopyOnPlan = true

	// We have fmt.Stringer implementations on lots of objects that hide
	// details that we very often want to see in tests, so we just disable
	// spew's use of String methods globally on the assumption that spew
	// usage implies an intent to see the raw values and ignore any
	// abstractions.
	spew.Config.DisableMethods = true

	os.Exit(m.Run())
}

func tempDir(t *testing.T) string {
	t.Helper()

	dir, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("err: %s", err)
	}

	return dir
}

// tempEnv lets you temporarily set an environment variable. It returns
// a function to defer to reset the old value.
// the old value that should be set via a defer.
func tempEnv(t *testing.T, k string, v string) func() {
	t.Helper()

	old, oldOk := os.LookupEnv(k)
	os.Setenv(k, v)
	return func() {
		if !oldOk {
			os.Unsetenv(k)
		} else {
			os.Setenv(k, old)
		}
	}
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

func testProviderFuncFixed(rp providers.Interface) providers.Factory {
	return func() (providers.Interface, error) {
		return rp, nil
	}
}

func testProvisionerFuncFixed(rp provisioners.Interface) ProvisionerFactory {
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

func instanceObjectIdForTests(obj *states.ResourceInstanceObject) string {
	v := obj.Value
	if v.IsNull() || !v.IsKnown() {
		return ""
	}
	idVal := v.GetAttr("id")
	if idVal.IsNull() || !idVal.IsKnown() {
		return ""
	}
	idVal, err := convert.Convert(idVal, cty.String)
	if err != nil {
		return "<invalid>" // placeholder value
	}
	return idVal.AsString()
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

const testTerraformInputProviderStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws
  bar = override
  foo = us-east-1
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider.aws
  bar = baz
  num = 2
  type = aws_instance
`

const testTerraformInputProviderOnlyStr = `
aws_instance.foo:
  ID = foo
  provider = provider.aws
  foo = us-west-2
  type = aws_instance
`

const testTerraformInputVarOnlyStr = `
aws_instance.foo:
  ID = foo
  provider = provider.aws
  foo = us-east-1
  type = aws_instance
`

const testTerraformInputVarOnlyUnsetStr = `
aws_instance.foo:
  ID = foo
  provider = provider.aws
  bar = baz
  foo = foovalue
  type = aws_instance
`

const testTerraformInputVarsStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws
  bar = override
  foo = us-east-1
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider.aws
  bar = baz
  num = 2
  type = aws_instance
`

const testTerraformApplyStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider.aws
  num = 2
  type = aws_instance
`

const testTerraformApplyDataBasicStr = `
data.null_data_source.testing:
  ID = yo
  provider = provider.null
`

const testTerraformApplyRefCountStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws
  foo = 3
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo.0:
  ID = foo
  provider = provider.aws
aws_instance.foo.1:
  ID = foo
  provider = provider.aws
aws_instance.foo.2:
  ID = foo
  provider = provider.aws
`

const testTerraformApplyProviderAliasStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws.bar
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider.aws
  num = 2
  type = aws_instance
`

const testTerraformApplyProviderAliasConfigStr = `
another_instance.bar:
  ID = foo
  provider = provider.another.two
another_instance.foo:
  ID = foo
  provider = provider.another
`

const testTerraformApplyEmptyModuleStr = `
<no state>
Outputs:

end = XXXX

module.child:
<no state>
Outputs:

aws_access_key = YYYYY
aws_route53_zone_id = XXXX
aws_secret_key = ZZZZ
`

const testTerraformApplyDependsCreateBeforeStr = `
aws_instance.lb:
  ID = baz
  provider = provider.aws
  instance = foo
  type = aws_instance

  Dependencies:
    aws_instance.web
aws_instance.web:
  ID = foo
  provider = provider.aws
  require_new = ami-new
  type = aws_instance
`

const testTerraformApplyCreateBeforeStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws
  require_new = xyz
  type = aws_instance
`

const testTerraformApplyCreateBeforeUpdateStr = `
aws_instance.bar:
  ID = bar
  provider = provider.aws
  foo = baz
  type = aws_instance
`

const testTerraformApplyCancelStr = `
aws_instance.foo:
  ID = foo
  provider = provider.aws
  value = 2
`

const testTerraformApplyComputeStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws
  foo = computed_value
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  provider = provider.aws
  compute = value
  compute_value = 1
  num = 2
  type = aws_instance
  value = computed_value
`

const testTerraformApplyCountDecStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws
  foo = bar
  type = aws_instance
aws_instance.foo.0:
  ID = bar
  provider = provider.aws
  foo = foo
  type = aws_instance
aws_instance.foo.1:
  ID = bar
  provider = provider.aws
  foo = foo
  type = aws_instance
`

const testTerraformApplyCountDecToOneStr = `
aws_instance.foo:
  ID = bar
  provider = provider.aws
  foo = foo
  type = aws_instance
`

const testTerraformApplyCountDecToOneCorruptedStr = `
aws_instance.foo:
  ID = bar
  provider = provider.aws
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
  provider = provider.aws
  foo = foo
  type = aws_instance
aws_instance.foo.0:
  ID = baz
  provider = provider.aws
  type = aws_instance
`

const testTerraformApplyCountVariableStr = `
aws_instance.foo.0:
  ID = foo
  provider = provider.aws
  foo = foo
  type = aws_instance
aws_instance.foo.1:
  ID = foo
  provider = provider.aws
  foo = foo
  type = aws_instance
`

const testTerraformApplyCountVariableRefStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws
  foo = 2
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo.0:
  ID = foo
  provider = provider.aws
aws_instance.foo.1:
  ID = foo
  provider = provider.aws
`
const testTerraformApplyForEachVariableStr = `
aws_instance.foo["b15c6d616d6143248c575900dff57325eb1de498"]:
  ID = foo
  provider = provider.aws
  foo = foo
  type = aws_instance
aws_instance.foo["c3de47d34b0a9f13918dd705c141d579dd6555fd"]:
  ID = foo
  provider = provider.aws
  foo = foo
  type = aws_instance
aws_instance.foo["e30a7edcc42a846684f2a4eea5f3cd261d33c46d"]:
  ID = foo
  provider = provider.aws
  foo = foo
  type = aws_instance
aws_instance.one["a"]:
  ID = foo
  provider = provider.aws
aws_instance.one["b"]:
  ID = foo
  provider = provider.aws
aws_instance.two["a"]:
  ID = foo
  provider = provider.aws

  Dependencies:
    aws_instance.one
aws_instance.two["b"]:
  ID = foo
  provider = provider.aws

  Dependencies:
    aws_instance.one`
const testTerraformApplyMinimalStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws
aws_instance.foo:
  ID = foo
  provider = provider.aws
`

const testTerraformApplyModuleStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider.aws
  num = 2
  type = aws_instance

module.child:
  aws_instance.baz:
    ID = foo
    provider = provider.aws
    foo = bar
    type = aws_instance
`

const testTerraformApplyModuleBoolStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws
  foo = true
  type = aws_instance

  Dependencies:
    module.child

module.child:
  <no state>
  Outputs:

  leader = true
`

const testTerraformApplyModuleDestroyOrderStr = `
<no state>
`

const testTerraformApplyMultiProviderStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws
  foo = bar
  type = aws_instance
do_instance.foo:
  ID = foo
  provider = provider.do
  num = 2
  type = do_instance
`

const testTerraformApplyModuleOnlyProviderStr = `
<no state>
module.child:
  aws_instance.foo:
    ID = foo
    provider = provider.aws
  test_instance.foo:
    ID = foo
    provider = provider.test
`

const testTerraformApplyModuleProviderAliasStr = `
<no state>
module.child:
  aws_instance.foo:
    ID = foo
    provider = module.child.provider.aws.eu
`

const testTerraformApplyModuleVarRefExistingStr = `
aws_instance.foo:
  ID = foo
  provider = provider.aws
  foo = bar

module.child:
  aws_instance.foo:
    ID = foo
    provider = provider.aws
    type = aws_instance
    value = bar
`

const testTerraformApplyOutputOrphanStr = `
<no state>
Outputs:

foo = bar
`

const testTerraformApplyOutputOrphanModuleStr = `
<no state>
module.child:
  <no state>
  Outputs:

  foo = bar
`

const testTerraformApplyProvisionerStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  provider = provider.aws
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
    provider = provider.aws
`

const testTerraformApplyProvisionerFailStr = `
aws_instance.bar: (tainted)
  ID = foo
  provider = provider.aws
aws_instance.foo:
  ID = foo
  provider = provider.aws
  num = 2
  type = aws_instance
`

const testTerraformApplyProvisionerFailCreateStr = `
aws_instance.bar: (tainted)
  ID = foo
  provider = provider.aws
`

const testTerraformApplyProvisionerFailCreateNoIdStr = `
<no state>
`

const testTerraformApplyProvisionerFailCreateBeforeDestroyStr = `
aws_instance.bar: (tainted) (1 deposed)
  ID = foo
  provider = provider.aws
  require_new = xyz
  type = aws_instance
  Deposed ID 1 = bar
`

const testTerraformApplyProvisionerResourceRefStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws
  num = 2
  type = aws_instance
`

const testTerraformApplyProvisionerSelfRefStr = `
aws_instance.foo:
  ID = foo
  provider = provider.aws
  foo = bar
  type = aws_instance
`

const testTerraformApplyProvisionerMultiSelfRefStr = `
aws_instance.foo.0:
  ID = foo
  provider = provider.aws
  foo = number 0
  type = aws_instance
aws_instance.foo.1:
  ID = foo
  provider = provider.aws
  foo = number 1
  type = aws_instance
aws_instance.foo.2:
  ID = foo
  provider = provider.aws
  foo = number 2
  type = aws_instance
`

const testTerraformApplyProvisionerMultiSelfRefSingleStr = `
aws_instance.foo.0:
  ID = foo
  provider = provider.aws
  foo = number 0
  type = aws_instance
aws_instance.foo.1:
  ID = foo
  provider = provider.aws
  foo = number 1
  type = aws_instance

  Dependencies:
    aws_instance.foo[0]
aws_instance.foo.2:
  ID = foo
  provider = provider.aws
  foo = number 2
  type = aws_instance

  Dependencies:
    aws_instance.foo[0]
`

const testTerraformApplyProvisionerDiffStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws
  foo = bar
  type = aws_instance
`

const testTerraformApplyDestroyStr = `
<no state>
`

const testTerraformApplyErrorStr = `
aws_instance.bar: (tainted)
  ID = bar
  provider = provider.aws

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  provider = provider.aws
  value = 2
`

const testTerraformApplyErrorCreateBeforeDestroyStr = `
aws_instance.bar:
  ID = bar
  provider = provider.aws
  require_new = abc
`

const testTerraformApplyErrorDestroyCreateBeforeDestroyStr = `
aws_instance.bar: (1 deposed)
  ID = foo
  provider = provider.aws
  require_new = xyz
  type = aws_instance
  Deposed ID 1 = bar
`

const testTerraformApplyErrorPartialStr = `
aws_instance.bar:
  ID = bar
  provider = provider.aws

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  provider = provider.aws
  value = 2
`

const testTerraformApplyResourceDependsOnModuleStr = `
aws_instance.a:
  ID = foo
  provider = provider.aws
  ami = parent
  type = aws_instance

  Dependencies:
    module.child

module.child:
  aws_instance.child:
    ID = foo
    provider = provider.aws
    ami = child
    type = aws_instance
`

const testTerraformApplyResourceDependsOnModuleDeepStr = `
aws_instance.a:
  ID = foo
  provider = provider.aws
  ami = parent
  type = aws_instance

  Dependencies:
    module.child

module.child.grandchild:
  aws_instance.c:
    ID = foo
    provider = provider.aws
    ami = grandchild
    type = aws_instance
`

const testTerraformApplyResourceDependsOnModuleInModuleStr = `
<no state>
module.child:
  aws_instance.b:
    ID = foo
    provider = provider.aws
    ami = child
    type = aws_instance

    Dependencies:
      module.grandchild
module.child.grandchild:
  aws_instance.c:
    ID = foo
    provider = provider.aws
    ami = grandchild
    type = aws_instance
`

const testTerraformApplyTaintStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws
  num = 2
  type = aws_instance
`

const testTerraformApplyTaintDepStr = `
aws_instance.bar:
  ID = bar
  provider = provider.aws
  foo = foo
  num = 2
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  provider = provider.aws
  num = 2
  type = aws_instance
`

const testTerraformApplyTaintDepRequireNewStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws
  foo = foo
  require_new = yes
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  provider = provider.aws
  num = 2
  type = aws_instance
`

const testTerraformApplyOutputStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider.aws
  num = 2
  type = aws_instance

Outputs:

foo_num = 2
`

const testTerraformApplyOutputAddStr = `
aws_instance.test.0:
  ID = foo
  provider = provider.aws
  foo = foo0
  type = aws_instance
aws_instance.test.1:
  ID = foo
  provider = provider.aws
  foo = foo1
  type = aws_instance

Outputs:

firstOutput = foo0
secondOutput = foo1
`

const testTerraformApplyOutputListStr = `
aws_instance.bar.0:
  ID = foo
  provider = provider.aws
  foo = bar
  type = aws_instance
aws_instance.bar.1:
  ID = foo
  provider = provider.aws
  foo = bar
  type = aws_instance
aws_instance.bar.2:
  ID = foo
  provider = provider.aws
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider.aws
  num = 2
  type = aws_instance

Outputs:

foo_num = [bar,bar,bar]
`

const testTerraformApplyOutputMultiStr = `
aws_instance.bar.0:
  ID = foo
  provider = provider.aws
  foo = bar
  type = aws_instance
aws_instance.bar.1:
  ID = foo
  provider = provider.aws
  foo = bar
  type = aws_instance
aws_instance.bar.2:
  ID = foo
  provider = provider.aws
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider.aws
  num = 2
  type = aws_instance

Outputs:

foo_num = bar,bar,bar
`

const testTerraformApplyOutputMultiIndexStr = `
aws_instance.bar.0:
  ID = foo
  provider = provider.aws
  foo = bar
  type = aws_instance
aws_instance.bar.1:
  ID = foo
  provider = provider.aws
  foo = bar
  type = aws_instance
aws_instance.bar.2:
  ID = foo
  provider = provider.aws
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  provider = provider.aws
  num = 2
  type = aws_instance

Outputs:

foo_num = bar
`

const testTerraformApplyUnknownAttrStr = `
aws_instance.foo: (tainted)
  ID = foo
  provider = provider.aws
  compute = unknown
  num = 2
  type = aws_instance
`

const testTerraformApplyVarsStr = `
aws_instance.bar:
  ID = foo
  provider = provider.aws
  bar = override
  baz = override
  foo = us-east-1
aws_instance.foo:
  ID = foo
  provider = provider.aws
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
  provider = provider.aws
  list.# = 2
  list.0 = Hello
  list.1 = World
  map.Baz = Foo
  map.Foo = Bar
  map.Hello = World
  string = baz
  type = aws_instance
`

const testTerraformPlanStr = `
DIFF:

CREATE: aws_instance.bar
  foo:  "" => "2"
  type: "" => "aws_instance"
CREATE: aws_instance.foo
  num:  "" => "2"
  type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanComputedIdStr = `
DIFF:

CREATE: aws_instance.bar
  foo:  "" => "<computed>"
  type: "" => "aws_instance"
CREATE: aws_instance.foo
  foo:  "" => "<computed>"
  num:  "" => "2"
  type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanCountIndexZeroStr = `
DIFF:

CREATE: aws_instance.foo
  foo:  "" => "0"
  type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanEmptyStr = `
DIFF:

CREATE: aws_instance.bar
CREATE: aws_instance.foo

STATE:

<no state>
`

const testTerraformPlanEscapedVarStr = `
DIFF:

CREATE: aws_instance.foo
  foo:  "" => "bar-${baz}"
  type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanModulesStr = `
DIFF:

CREATE: aws_instance.bar
  foo:  "" => "2"
  type: "" => "aws_instance"
CREATE: aws_instance.foo
  num:  "" => "2"
  type: "" => "aws_instance"

module.child:
  CREATE: aws_instance.foo
    num:  "" => "2"
    type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanModuleCycleStr = `
DIFF:

CREATE: aws_instance.b
CREATE: aws_instance.c
  some_input: "" => "<computed>"
  type:       "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanModuleInputStr = `
DIFF:

CREATE: aws_instance.bar
  foo:  "" => "2"
  type: "" => "aws_instance"

module.child:
  CREATE: aws_instance.foo
    foo:  "" => "42"
    type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanModuleInputComputedStr = `
DIFF:

CREATE: aws_instance.bar
  compute:       "" => "foo"
  compute_value: "" => "<computed>"
  foo:           "" => "<computed>"
  type:          "" => "aws_instance"

module.child:
  CREATE: aws_instance.foo
    foo:  "" => "<computed>"
    type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanModuleVarIntStr = `
DIFF:

module.child:
  CREATE: aws_instance.foo
    num:  "" => "2"
    type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanMultipleTaintStr = `
DIFF:

DESTROY/CREATE: aws_instance.bar
  foo:  "" => "2"
  type: "" => "aws_instance"

STATE:

aws_instance.bar: (2 tainted)
  ID = <not created>
  Tainted ID 1 = baz
  Tainted ID 2 = zip
aws_instance.foo:
  ID = bar
  num = 2
`

const testTerraformPlanVarMultiCountOneStr = `
DIFF:

CREATE: aws_instance.bar
  foo:  "" => "2"
  type: "" => "aws_instance"
CREATE: aws_instance.foo
  num:  "" => "2"
  type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformInputHCL = `
hcl_instance.hcltest:
  ID = foo
  provider = provider.hcl
  bar.w = z
  bar.x = y
  foo.# = 2
  foo.0 = a
  foo.1 = b
  type = hcl_instance
`

const testTerraformRefreshDataRefDataStr = `
data.null_data_source.bar:
  ID = foo
  provider = provider.null
  bar = yes

  Dependencies:
    data.null_data_source.foo
data.null_data_source.foo:
  ID = foo
  provider = provider.null
  foo = yes
`
