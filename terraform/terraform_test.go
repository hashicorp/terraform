package terraform

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/config"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./test-fixtures"

func testConfig(t *testing.T, name string) *config.Config {
	c, err := config.Load(filepath.Join(fixtureDir, name, "main.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return c
}

func testProviderFuncFixed(rp ResourceProvider) ResourceProviderFactory {
	return func() (ResourceProvider, error) {
		return rp, nil
	}
}

// HookRecordApplyOrder is a test hook that records the order of applies
// by recording the PreApply event.
type HookRecordApplyOrder struct {
	NilHook

	Active bool

	IDs    []string
	States []*ResourceState
	Diffs  []*ResourceDiff

	l sync.Mutex
}

func (h *HookRecordApplyOrder) PreApply(
	id string,
	s *ResourceState,
	d *ResourceDiff) (HookAction, error) {
	if h.Active {
		h.l.Lock()
		defer h.l.Unlock()

		h.IDs = append(h.IDs, id)
		h.Diffs = append(h.Diffs, d)
		h.States = append(h.States, s)
	}

	return HookActionContinue, nil
}

// Below are all the constant strings that are the expected output for
// various tests.

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
aws_instance.foo:
  ID = foo
  dynamical = computed_dynamical
  num = 2
  type = aws_instance
`

const testTerraformApplyDestroyStr = `
<no state>
`

const testTerraformApplyErrorStr = `
aws_instance.foo:
  ID = foo
  num = 2
`

const testTerraformApplyErrorPartialStr = `
aws_instance.bar:
  ID = bar
aws_instance.foo:
  ID = foo
  num = 2
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
  foo = bar
  type = aws_instance
aws_instance.foo:
  ID = foo
  num = 2
  type = aws_instance
`

const testTerraformPlanStr = `
DIFF:

UPDATE: aws_instance.bar
  foo:  "" => "2"
  type: "" => "aws_instance"
UPDATE: aws_instance.foo
  num:  "" => "2"
  type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanComputedStr = `
DIFF:

UPDATE: aws_instance.bar
  foo:  "" => "<computed>"
  type: "" => "aws_instance"
UPDATE: aws_instance.foo
  id:   "" => "<computed>"
  num:  "" => "2"
  type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanCountStr = `
DIFF:

UPDATE: aws_instance.bar
  foo:  "" => "foo,foo,foo,foo,foo"
  type: "" => "aws_instance"
UPDATE: aws_instance.foo.0
  foo:  "" => "foo"
  type: "" => "aws_instance"
UPDATE: aws_instance.foo.1
  foo:  "" => "foo"
  type: "" => "aws_instance"
UPDATE: aws_instance.foo.2
  foo:  "" => "foo"
  type: "" => "aws_instance"
UPDATE: aws_instance.foo.3
  foo:  "" => "foo"
  type: "" => "aws_instance"
UPDATE: aws_instance.foo.4
  foo:  "" => "foo"
  type: "" => "aws_instance"

STATE:

<no state>
`

const testTerraformPlanCountDecreaseStr = `
DIFF:

UPDATE: aws_instance.bar
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

UPDATE: aws_instance.bar
  foo:  "" => "bar"
  type: "" => "aws_instance"
UPDATE: aws_instance.foo.1
  foo:  "" => "foo"
  type: "" => "aws_instance"
UPDATE: aws_instance.foo.2
  foo:  "" => "foo"
  type: "" => "aws_instance"

STATE:

aws_instance.foo:
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

const testTerraformPlanOrphanStr = `
DIFF:

DESTROY: aws_instance.baz
UPDATE: aws_instance.foo
  num:  "" => "2"
  type: "" => "aws_instance"

STATE:

aws_instance.baz:
  ID = bar
`

const testTerraformPlanStateStr = `
DIFF:

UPDATE: aws_instance.bar
  foo:  "" => "2"
  type: "" => "aws_instance"
UPDATE: aws_instance.foo
  num:  "" => "2"
  type: "" => ""

STATE:

aws_instance.foo:
  ID = bar
`
