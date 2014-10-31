package terraform

import (
	"bytes"
	"crypto/sha1"
	"encoding/gob"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./test-fixtures"

func checksumStruct(t *testing.T, i interface{}) string {
	// TODO(mitchellh): write a library to do this because gob is not
	// deterministic in order
	return "foo"

	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(i); err != nil {
		t.Fatalf("err: %s", err)
	}

	sum := sha1.Sum(buf.Bytes())
	return hex.EncodeToString(sum[:])
}

func tempDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("err: %s", err)
	}

	return dir
}

func testConfig(t *testing.T, name string) *config.Config {
	c, err := config.Load(filepath.Join(fixtureDir, name, "main.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return c
}

func testModule(t *testing.T, name string) *module.Tree {
	mod, err := module.NewTreeModule("", filepath.Join(fixtureDir, name))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	s := &module.FolderStorage{StorageDir: tempDir(t)}
	if err := mod.Load(s, module.GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	return mod
}

func testProviderFuncFixed(rp ResourceProvider) ResourceProviderFactory {
	return func() (ResourceProvider, error) {
		return rp, nil
	}
}

func testProvisionerFuncFixed(rp ResourceProvisioner) ResourceProvisionerFactory {
	return func() (ResourceProvisioner, error) {
		return rp, nil
	}
}

// HookRecordApplyOrder is a test hook that records the order of applies
// by recording the PreApply event.
type HookRecordApplyOrder struct {
	NilHook

	Active bool

	IDs    []string
	States []*InstanceState
	Diffs  []*InstanceDiff

	l sync.Mutex
}

func (h *HookRecordApplyOrder) PreApply(
	info *InstanceInfo,
	s *InstanceState,
	d *InstanceDiff) (HookAction, error) {
	if h.Active {
		h.l.Lock()
		defer h.l.Unlock()

		h.IDs = append(h.IDs, info.Id)
		h.Diffs = append(h.Diffs, d)
		h.States = append(h.States, s)
	}

	return HookActionContinue, nil
}

// Below are all the constant strings that are the expected output for
// various tests.

const testTerraformInputProviderStr = `
aws_instance.bar:
  ID = foo
  bar = override
  foo = us-east-1
  type = aws_instance
aws_instance.foo:
  ID = foo
  bar = baz
  num = 2
  type = aws_instance
`

const testTerraformInputProviderOnlyStr = `
aws_instance.foo:
  ID = foo
  foo = us-west-2
  type = aws_instance
`

const testTerraformInputVarOnlyStr = `
aws_instance.foo:
  ID = foo
  foo = us-east-1
  type = aws_instance
`

const testTerraformInputVarsStr = `
aws_instance.bar:
  ID = foo
  bar = override
  foo = us-east-1
  type = aws_instance
aws_instance.foo:
  ID = foo
  bar = baz
  num = 2
  type = aws_instance
`

const testTerraformApplyStr = `
aws_instance.bar:
  ID = foo
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  num = 2
  type = aws_instance
`

const testTerraformApplyDependsCreateBeforeStr = `
aws_instance.lb:
  ID = foo
  instance = foo
  type = aws_instance

  Dependencies:
    aws_instance.web
aws_instance.web:
  ID = foo
  require_new = ami-new
  type = aws_instance
`

const testTerraformApplyCreateBeforeStr = `
aws_instance.bar:
  ID = foo
  require_new = xyz
  type = aws_instance
`

const testTerraformApplyCancelStr = `
aws_instance.foo:
  ID = foo
  num = 2
`

const testTerraformApplyComputeStr = `
aws_instance.bar:
  ID = foo
  foo = computed_dynamical
  type = aws_instance

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  dynamical = computed_dynamical
  num = 2
  type = aws_instance
`

const testTerraformApplyCountDecStr = `
aws_instance.foo.0:
  ID = bar
  foo = foo
  type = aws_instance
aws_instance.foo.1:
  ID = bar
  foo = foo
  type = aws_instance
`

const testTerraformApplyCountDecToOneStr = `
aws_instance.foo.0:
  ID = bar
  foo = foo
  type = aws_instance
`

const testTerraformApplyCountTaintedStr = `
<no state>
`

const testTerraformApplyCountVariableStr = `
aws_instance.foo.0:
  ID = foo
  foo = foo
  type = aws_instance
aws_instance.foo.1:
  ID = foo
  foo = foo
  type = aws_instance
`

const testTerraformApplyMinimalStr = `
aws_instance.bar:
  ID = foo
aws_instance.foo:
  ID = foo
`

const testTerraformApplyModuleStr = `
aws_instance.bar:
  ID = foo
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  num = 2
  type = aws_instance

module.child:
  aws_instance.baz:
    ID = foo
    foo = bar
    type = aws_instance
`

const testTerraformApplyProvisionerStr = `
aws_instance.bar:
  ID = foo

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  dynamical = computed_dynamical
  num = 2
  type = aws_instance
`

const testTerraformApplyProvisionerFailStr = `
aws_instance.bar: (1 tainted)
  ID = <not created>
  Tainted ID 1 = foo
aws_instance.foo:
  ID = foo
  num = 2
  type = aws_instance
`

const testTerraformApplyProvisionerFailCreateStr = `
aws_instance.bar: (1 tainted)
  ID = <not created>
  Tainted ID 1 = foo
`

const testTerraformApplyProvisionerFailCreateNoIdStr = `
<no state>
`

const testTerraformApplyProvisionerFailCreateBeforeDestroyStr = `
aws_instance.bar: (1 tainted)
  ID = bar
  require_new = abc
  Tainted ID 1 = foo
`

const testTerraformApplyProvisionerResourceRefStr = `
aws_instance.bar:
  ID = foo
  num = 2
  type = aws_instance
`

const testTerraformApplyProvisionerDiffStr = `
aws_instance.bar:
  ID = foo
  foo = bar
  type = aws_instance
`

const testTerraformApplyDestroyStr = `
<no state>
`

const testTerraformApplyErrorStr = `
aws_instance.bar:
  ID = bar

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  num = 2
`

const testTerraformApplyErrorCreateBeforeDestroyStr = `
aws_instance.bar:
  ID = bar
  require_new = abc
`

const testTerraformApplyErrorDestroyCreateBeforeDestroyStr = `
aws_instance.bar: (1 tainted)
  ID = foo
  Tainted ID 1 = bar
`

const testTerraformApplyErrorPartialStr = `
aws_instance.bar:
  ID = bar

  Dependencies:
    aws_instance.foo
aws_instance.foo:
  ID = foo
  num = 2
`

const testTerraformApplyTaintStr = `
aws_instance.bar:
  ID = foo
  num = 2
  type = aws_instance
`

const testTerraformApplyOutputStr = `
aws_instance.bar:
  ID = foo
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  num = 2
  type = aws_instance

Outputs:

foo_num = 2
`

const testTerraformApplyOutputMultiStr = `
aws_instance.bar.0:
  ID = foo
  foo = bar
  type = aws_instance
aws_instance.bar.1:
  ID = foo
  foo = bar
  type = aws_instance
aws_instance.bar.2:
  ID = foo
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  num = 2
  type = aws_instance

Outputs:

foo_num = bar,bar,bar
`

const testTerraformApplyOutputMultiIndexStr = `
aws_instance.bar.0:
  ID = foo
  foo = bar
  type = aws_instance
aws_instance.bar.1:
  ID = foo
  foo = bar
  type = aws_instance
aws_instance.bar.2:
  ID = foo
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  num = 2
  type = aws_instance

Outputs:

foo_num = bar
`

const testTerraformApplyUnknownAttrStr = `
aws_instance.foo:
  ID = foo
  num = 2
  type = aws_instance
`

const testTerraformApplyVarsStr = `
aws_instance.bar:
  ID = foo
  bar = foo
  baz = override
  foo = us-west-2
  type = aws_instance
aws_instance.foo:
  ID = foo
  bar = baz
  num = 2
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

const testTerraformPlanComputedStr = `
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

const testTerraformPlanComputedListStr = `
DIFF:

CREATE: aws_instance.bar
  foo:  "" => "<computed>"
  type: "" => "aws_instance"
CREATE: aws_instance.foo
  list.#: "" => "<computed>"
  num:    "" => "2"
  type:   "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanCountStr = `
DIFF:

CREATE: aws_instance.bar
  foo:  "" => "foo,foo,foo,foo,foo"
  type: "" => "aws_instance"
CREATE: aws_instance.foo.0
  foo:  "" => "foo"
  type: "" => "aws_instance"
CREATE: aws_instance.foo.1
  foo:  "" => "foo"
  type: "" => "aws_instance"
CREATE: aws_instance.foo.2
  foo:  "" => "foo"
  type: "" => "aws_instance"
CREATE: aws_instance.foo.3
  foo:  "" => "foo"
  type: "" => "aws_instance"
CREATE: aws_instance.foo.4
  foo:  "" => "foo"
  type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanCountIndexStr = `
DIFF:

CREATE: aws_instance.foo.0
  foo:  "" => "0"
  type: "" => "aws_instance"
CREATE: aws_instance.foo.1
  foo:  "" => "1"
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

const testTerraformPlanCountOneIndexStr = `
DIFF:

CREATE: aws_instance.bar
  foo:  "" => "foo"
  type: "" => "aws_instance"
CREATE: aws_instance.foo
  foo:  "" => "foo"
  type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanCountZeroStr = `
DIFF:

CREATE: aws_instance.bar
  foo:  "" => ""
  type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanCountVarStr = `
DIFF:

CREATE: aws_instance.bar
  foo:  "" => "foo,foo,foo"
  type: "" => "aws_instance"
CREATE: aws_instance.foo.0
  foo:  "" => "foo"
  type: "" => "aws_instance"
CREATE: aws_instance.foo.1
  foo:  "" => "foo"
  type: "" => "aws_instance"
CREATE: aws_instance.foo.2
  foo:  "" => "foo"
  type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanCountDecreaseStr = `
DIFF:

CREATE: aws_instance.bar
  foo:  "" => "bar"
  type: "" => "aws_instance"
DESTROY: aws_instance.foo.1
DESTROY: aws_instance.foo.2

STATE:

aws_instance.foo.0:
  ID = bar
  foo = foo
  type = aws_instance
aws_instance.foo.1:
  ID = bar
aws_instance.foo.2:
  ID = bar
`

const testTerraformPlanCountIncreaseStr = `
DIFF:

CREATE: aws_instance.bar
  foo:  "" => "bar"
  type: "" => "aws_instance"
CREATE: aws_instance.foo.1
  foo:  "" => "foo"
  type: "" => "aws_instance"
CREATE: aws_instance.foo.2
  foo:  "" => "foo"
  type: "" => "aws_instance"

STATE:

aws_instance.foo:
  ID = bar
  foo = foo
  type = aws_instance
`

const testTerraformPlanCountIncreaseFromOneStr = `
DIFF:

CREATE: aws_instance.bar
  foo:  "" => "bar"
  type: "" => "aws_instance"
CREATE: aws_instance.foo.1
  foo:  "" => "foo"
  type: "" => "aws_instance"
CREATE: aws_instance.foo.2
  foo:  "" => "foo"
  type: "" => "aws_instance"

STATE:

aws_instance.foo.0:
  ID = bar
  foo = foo
  type = aws_instance
`

const testTerraformPlanDestroyStr = `
DIFF:

DESTROY: aws_instance.one
DESTROY: aws_instance.two

STATE:

aws_instance.one:
  ID = bar
aws_instance.two:
  ID = baz
`

const testTerraformPlanDiffVarStr = `
DIFF:

CREATE: aws_instance.bar
  num:  "" => "3"
  type: "" => "aws_instance"
UPDATE: aws_instance.foo
  num: "2" => "3"

STATE:

aws_instance.foo:
  ID = bar
  num = 2
`

const testTerraformPlanEmptyStr = `
DIFF:

CREATE: aws_instance.bar
CREATE: aws_instance.foo

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

const testTerraformPlanModuleDestroyStr = `
DIFF:

DESTROY: aws_instance.foo

module.child:
  DESTROY: aws_instance.foo

STATE:

aws_instance.foo:
  ID = bar

module.child:
  aws_instance.foo:
    ID = bar
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
  foo:  "" => "<computed>"
  type: "" => "aws_instance"

module.child:
  CREATE: aws_instance.foo
    foo:  "" => "<computed>"
    type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanModuleInputVarStr = `
DIFF:

CREATE: aws_instance.bar
  foo:  "" => "2"
  type: "" => "aws_instance"

module.child:
  CREATE: aws_instance.foo
    foo:  "" => "52"
    type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanModuleOrphansStr = `
DIFF:

CREATE: aws_instance.foo
  num:  "" => "2"
  type: "" => "aws_instance"

module.child:
  DESTROY: aws_instance.foo

STATE:

module.child:
  aws_instance.foo:
    ID = baz
`

const testTerraformPlanModuleVarStr = `
DIFF:

CREATE: aws_instance.bar
  foo:  "" => "2"
  type: "" => "aws_instance"

module.child:
  CREATE: aws_instance.foo
    num:  "" => "2"
    type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanModuleVarComputedStr = `
DIFF:

CREATE: aws_instance.bar
  foo:  "" => "<computed>"
  type: "" => "aws_instance"

module.child:
  CREATE: aws_instance.foo
    foo:  "" => "<computed>"
    type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanOrphanStr = `
DIFF:

DESTROY: aws_instance.baz
CREATE: aws_instance.foo
  num:  "" => "2"
  type: "" => "aws_instance"

STATE:

aws_instance.baz:
  ID = bar
`

const testTerraformPlanStateStr = `
DIFF:

CREATE: aws_instance.bar
  foo:  "" => "2"
  type: "" => "aws_instance"
UPDATE: aws_instance.foo
  num:  "" => "2"
  type: "" => "aws_instance"

STATE:

aws_instance.foo:
  ID = bar
`

const testTerraformPlanTaintStr = `
DIFF:

DESTROY/CREATE: aws_instance.bar
  foo:  "" => "2"
  type: "" => "aws_instance"

STATE:

aws_instance.bar: (1 tainted)
  ID = <not created>
  Tainted ID 1 = baz
aws_instance.foo:
  ID = bar
  num = 2
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

const testTerraformPlanPathVarStr = `
DIFF:

CREATE: aws_instance.foo
  cwd:    "" => "%s/barpath"
  module: "" => "%s/foopath"
  root:   "" => "%s/barpath"
  type:   "" => "aws_instance"

STATE:

<no state>
`
