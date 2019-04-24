package resource

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/terraform"
)

func init() {
	testTesting = true

	// TODO: Remove when we remove the guard on id checks
	if err := os.Setenv("TF_ACC_IDONLY", "1"); err != nil {
		panic(err)
	}

	if err := os.Setenv(TestEnvVar, "1"); err != nil {
		panic(err)
	}
}

// wrap the mock provider to implement TestProvider
type resetProvider struct {
	*terraform.MockResourceProvider
	mu              sync.Mutex
	TestResetCalled bool
	TestResetError  error
}

func (p *resetProvider) TestReset() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.TestResetCalled = true
	return p.TestResetError
}

func TestParallelTest(t *testing.T) {
	mt := new(mockT)
	ParallelTest(mt, TestCase{})

	if !mt.ParallelCalled {
		t.Fatal("Parallel() not called")
	}
}

func TestTest(t *testing.T) {
	t.Skip("test requires new provider implementation")

	mp := &resetProvider{
		MockResourceProvider: testProvider(),
	}

	mp.DiffReturn = nil

	mp.ApplyFn = func(
		info *terraform.InstanceInfo,
		state *terraform.InstanceState,
		diff *terraform.InstanceDiff) (*terraform.InstanceState, error) {
		if !diff.Destroy {
			return &terraform.InstanceState{
				ID: "foo",
			}, nil
		}

		return nil, nil
	}

	var refreshCount int32
	mp.RefreshFn = func(*terraform.InstanceInfo, *terraform.InstanceState) (*terraform.InstanceState, error) {
		atomic.AddInt32(&refreshCount, 1)
		return &terraform.InstanceState{ID: "foo"}, nil
	}

	checkDestroy := false
	checkStep := false

	checkDestroyFn := func(*terraform.State) error {
		checkDestroy = true
		return nil
	}

	checkStepFn := func(s *terraform.State) error {
		checkStep = true

		rs, ok := s.RootModule().Resources["test_instance.foo"]
		if !ok {
			t.Error("test_instance.foo is not present")
			return nil
		}
		is := rs.Primary
		if is.ID != "foo" {
			t.Errorf("bad check ID: %s", is.ID)
		}

		return nil
	}

	mt := new(mockT)
	Test(mt, TestCase{
		Providers: map[string]terraform.ResourceProvider{
			"test": mp,
		},
		CheckDestroy: checkDestroyFn,
		Steps: []TestStep{
			TestStep{
				Config: testConfigStr,
				Check:  checkStepFn,
			},
		},
	})

	if mt.failed() {
		t.Fatalf("test failed: %s", mt.failMessage())
	}
	if mt.ParallelCalled {
		t.Fatal("Parallel() called")
	}
	if !checkStep {
		t.Fatal("didn't call check for step")
	}
	if !checkDestroy {
		t.Fatal("didn't call check for destroy")
	}
	if !mp.TestResetCalled {
		t.Fatal("didn't call TestReset")
	}
}

func TestTest_plan_only(t *testing.T) {
	t.Skip("test requires new provider implementation")

	mp := testProvider()
	mp.ApplyReturn = &terraform.InstanceState{
		ID: "foo",
	}

	checkDestroy := false

	checkDestroyFn := func(*terraform.State) error {
		checkDestroy = true
		return nil
	}

	mt := new(mockT)
	Test(mt, TestCase{
		Providers: map[string]terraform.ResourceProvider{
			"test": mp,
		},
		CheckDestroy: checkDestroyFn,
		Steps: []TestStep{
			TestStep{
				Config:             testConfigStr,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})

	if !mt.failed() {
		t.Fatal("test should've failed")
	}

	expected := `Step 0 error: After applying this step, the plan was not empty:

DIFF:

CREATE: test_instance.foo
  foo: "" => "bar"

STATE:

<no state>`

	if mt.failMessage() != expected {
		t.Fatalf("Expected message: %s\n\ngot:\n\n%s", expected, mt.failMessage())
	}

	if !checkDestroy {
		t.Fatal("didn't call check for destroy")
	}
}

func TestTest_idRefresh(t *testing.T) {
	t.Skip("test requires new provider implementation")

	// Refresh count should be 3:
	//   1.) initial Ref/Plan/Apply
	//   2.) post Ref/Plan/Apply for plan-check
	//   3.) id refresh check
	var expectedRefresh int32 = 3

	mp := testProvider()
	mp.DiffReturn = nil

	mp.ApplyFn = func(
		info *terraform.InstanceInfo,
		state *terraform.InstanceState,
		diff *terraform.InstanceDiff) (*terraform.InstanceState, error) {
		if !diff.Destroy {
			return &terraform.InstanceState{
				ID: "foo",
			}, nil
		}

		return nil, nil
	}

	var refreshCount int32
	mp.RefreshFn = func(*terraform.InstanceInfo, *terraform.InstanceState) (*terraform.InstanceState, error) {
		atomic.AddInt32(&refreshCount, 1)
		return &terraform.InstanceState{ID: "foo"}, nil
	}

	mt := new(mockT)
	Test(mt, TestCase{
		IDRefreshName: "test_instance.foo",
		Providers: map[string]terraform.ResourceProvider{
			"test": mp,
		},
		Steps: []TestStep{
			TestStep{
				Config: testConfigStr,
			},
		},
	})

	if mt.failed() {
		t.Fatalf("test failed: %s", mt.failMessage())
	}

	// See declaration of expectedRefresh for why that number
	if refreshCount != expectedRefresh {
		t.Fatalf("bad refresh count: %d", refreshCount)
	}
}

func TestTest_idRefreshCustomName(t *testing.T) {
	t.Skip("test requires new provider implementation")

	// Refresh count should be 3:
	//   1.) initial Ref/Plan/Apply
	//   2.) post Ref/Plan/Apply for plan-check
	//   3.) id refresh check
	var expectedRefresh int32 = 3

	mp := testProvider()
	mp.DiffReturn = nil

	mp.ApplyFn = func(
		info *terraform.InstanceInfo,
		state *terraform.InstanceState,
		diff *terraform.InstanceDiff) (*terraform.InstanceState, error) {
		if !diff.Destroy {
			return &terraform.InstanceState{
				ID: "foo",
			}, nil
		}

		return nil, nil
	}

	var refreshCount int32
	mp.RefreshFn = func(*terraform.InstanceInfo, *terraform.InstanceState) (*terraform.InstanceState, error) {
		atomic.AddInt32(&refreshCount, 1)
		return &terraform.InstanceState{ID: "foo"}, nil
	}

	mt := new(mockT)
	Test(mt, TestCase{
		IDRefreshName: "test_instance.foo",
		Providers: map[string]terraform.ResourceProvider{
			"test": mp,
		},
		Steps: []TestStep{
			TestStep{
				Config: testConfigStr,
			},
		},
	})

	if mt.failed() {
		t.Fatalf("test failed: %s", mt.failMessage())
	}

	// See declaration of expectedRefresh for why that number
	if refreshCount != expectedRefresh {
		t.Fatalf("bad refresh count: %d", refreshCount)
	}
}

func TestTest_idRefreshFail(t *testing.T) {
	t.Skip("test requires new provider implementation")

	// Refresh count should be 3:
	//   1.) initial Ref/Plan/Apply
	//   2.) post Ref/Plan/Apply for plan-check
	//   3.) id refresh check
	var expectedRefresh int32 = 3

	mp := testProvider()
	mp.DiffReturn = nil

	mp.ApplyFn = func(
		info *terraform.InstanceInfo,
		state *terraform.InstanceState,
		diff *terraform.InstanceDiff) (*terraform.InstanceState, error) {
		if !diff.Destroy {
			return &terraform.InstanceState{
				ID: "foo",
			}, nil
		}

		return nil, nil
	}

	var refreshCount int32
	mp.RefreshFn = func(*terraform.InstanceInfo, *terraform.InstanceState) (*terraform.InstanceState, error) {
		atomic.AddInt32(&refreshCount, 1)
		if atomic.LoadInt32(&refreshCount) == expectedRefresh-1 {
			return &terraform.InstanceState{
				ID:         "foo",
				Attributes: map[string]string{"foo": "bar"},
			}, nil
		} else if atomic.LoadInt32(&refreshCount) < expectedRefresh {
			return &terraform.InstanceState{ID: "foo"}, nil
		} else {
			return nil, nil
		}
	}

	mt := new(mockT)
	Test(mt, TestCase{
		IDRefreshName: "test_instance.foo",
		Providers: map[string]terraform.ResourceProvider{
			"test": mp,
		},
		Steps: []TestStep{
			TestStep{
				Config: testConfigStr,
			},
		},
	})

	if !mt.failed() {
		t.Fatal("test didn't fail")
	}
	t.Logf("failure reason: %s", mt.failMessage())

	// See declaration of expectedRefresh for why that number
	if refreshCount != expectedRefresh {
		t.Fatalf("bad refresh count: %d", refreshCount)
	}
}

func TestTest_empty(t *testing.T) {
	t.Skip("test requires new provider implementation")

	destroyCalled := false
	checkDestroyFn := func(*terraform.State) error {
		destroyCalled = true
		return nil
	}

	mt := new(mockT)
	Test(mt, TestCase{
		CheckDestroy: checkDestroyFn,
	})

	if mt.failed() {
		t.Fatal("test failed")
	}
	if destroyCalled {
		t.Fatal("should not call check destroy if there is no steps")
	}
}

func TestTest_noEnv(t *testing.T) {
	t.Skip("test requires new provider implementation")

	// Unset the variable
	if err := os.Setenv(TestEnvVar, ""); err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Setenv(TestEnvVar, "1")

	mt := new(mockT)
	Test(mt, TestCase{})

	if !mt.SkipCalled {
		t.Fatal("skip not called")
	}
}

func TestTest_preCheck(t *testing.T) {
	t.Skip("test requires new provider implementation")

	called := false

	mt := new(mockT)
	Test(mt, TestCase{
		PreCheck: func() { called = true },
	})

	if !called {
		t.Fatal("precheck should be called")
	}
}

func TestTest_skipFunc(t *testing.T) {
	t.Skip("test requires new provider implementation")

	preCheckCalled := false
	skipped := false

	mp := testProvider()
	mp.ApplyReturn = &terraform.InstanceState{
		ID: "foo",
	}

	checkStepFn := func(*terraform.State) error {
		return fmt.Errorf("error")
	}

	mt := new(mockT)
	Test(mt, TestCase{
		Providers: map[string]terraform.ResourceProvider{
			"test": mp,
		},
		PreCheck: func() { preCheckCalled = true },
		Steps: []TestStep{
			{
				Config:   testConfigStr,
				Check:    checkStepFn,
				SkipFunc: func() (bool, error) { skipped = true; return true, nil },
			},
		},
	})

	if mt.failed() {
		t.Fatal("Expected check to be skipped")
	}

	if !preCheckCalled {
		t.Fatal("precheck should be called")
	}
	if !skipped {
		t.Fatal("SkipFunc should be called")
	}
}

func TestTest_stepError(t *testing.T) {
	t.Skip("test requires new provider implementation")

	mp := testProvider()
	mp.ApplyReturn = &terraform.InstanceState{
		ID: "foo",
	}

	checkDestroy := false

	checkDestroyFn := func(*terraform.State) error {
		checkDestroy = true
		return nil
	}

	checkStepFn := func(*terraform.State) error {
		return fmt.Errorf("error")
	}

	mt := new(mockT)
	Test(mt, TestCase{
		Providers: map[string]terraform.ResourceProvider{
			"test": mp,
		},
		CheckDestroy: checkDestroyFn,
		Steps: []TestStep{
			TestStep{
				Config: testConfigStr,
				Check:  checkStepFn,
			},
		},
	})

	if !mt.failed() {
		t.Fatal("test should've failed")
	}
	expected := "Step 0 error: Check failed: error"
	if mt.failMessage() != expected {
		t.Fatalf("Expected message: %s\n\ngot:\n\n%s", expected, mt.failMessage())
	}

	if !checkDestroy {
		t.Fatal("didn't call check for destroy")
	}
}

func TestTest_factoryError(t *testing.T) {
	resourceFactoryError := fmt.Errorf("resource factory error")

	factory := func() (terraform.ResourceProvider, error) {
		return nil, resourceFactoryError
	}

	mt := new(mockT)
	Test(mt, TestCase{
		ProviderFactories: map[string]terraform.ResourceProviderFactory{
			"test": factory,
		},
		Steps: []TestStep{
			TestStep{
				ExpectError: regexp.MustCompile("resource factory error"),
			},
		},
	})

	if !mt.failed() {
		t.Fatal("test should've failed")
	}
}

func TestTest_resetError(t *testing.T) {
	t.Skip("test requires new provider implementation")

	mp := &resetProvider{
		MockResourceProvider: testProvider(),
		TestResetError:       fmt.Errorf("provider reset error"),
	}

	mt := new(mockT)
	Test(mt, TestCase{
		Providers: map[string]terraform.ResourceProvider{
			"test": mp,
		},
		Steps: []TestStep{
			TestStep{
				ExpectError: regexp.MustCompile("provider reset error"),
			},
		},
	})

	if !mt.failed() {
		t.Fatal("test should've failed")
	}
}

func TestTest_expectError(t *testing.T) {
	t.Skip("test requires new provider implementation")

	cases := []struct {
		name     string
		planErr  bool
		applyErr bool
		badErr   bool
	}{
		{
			name:     "successful apply",
			planErr:  false,
			applyErr: false,
		},
		{
			name:     "bad plan",
			planErr:  true,
			applyErr: false,
		},
		{
			name:     "bad apply",
			planErr:  false,
			applyErr: true,
		},
		{
			name:     "bad plan, bad err",
			planErr:  true,
			applyErr: false,
			badErr:   true,
		},
		{
			name:     "bad apply, bad err",
			planErr:  false,
			applyErr: true,
			badErr:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mp := testProvider()
			expectedText := "test provider error"
			var errText string
			if tc.badErr {
				errText = "wrong provider error"
			} else {
				errText = expectedText
			}
			noErrText := "no error received, but expected a match to"
			if tc.planErr {
				mp.DiffReturnError = errors.New(errText)
			}
			if tc.applyErr {
				mp.ApplyReturnError = errors.New(errText)
			}
			mt := new(mockT)
			Test(mt, TestCase{
				Providers: map[string]terraform.ResourceProvider{
					"test": mp,
				},
				Steps: []TestStep{
					TestStep{
						Config:             testConfigStr,
						ExpectError:        regexp.MustCompile(expectedText),
						Check:              func(*terraform.State) error { return nil },
						ExpectNonEmptyPlan: true,
					},
				},
			},
			)
			if mt.FatalCalled {
				t.Fatalf("fatal: %+v", mt.FatalArgs)
			}
			switch {
			case len(mt.ErrorArgs) < 1 && !tc.planErr && !tc.applyErr:
				t.Fatalf("expected error, got none")
			case !tc.planErr && !tc.applyErr:
				for _, e := range mt.ErrorArgs {
					if regexp.MustCompile(noErrText).MatchString(fmt.Sprintf("%v", e)) {
						return
					}
				}
				t.Fatalf("expected error to match %s, got %+v", noErrText, mt.ErrorArgs)
			case tc.badErr:
				for _, e := range mt.ErrorArgs {
					if regexp.MustCompile(expectedText).MatchString(fmt.Sprintf("%v", e)) {
						return
					}
				}
				t.Fatalf("expected error to match %s, got %+v", expectedText, mt.ErrorArgs)
			}
		})
	}
}

func TestComposeAggregateTestCheckFunc(t *testing.T) {
	check1 := func(s *terraform.State) error {
		return errors.New("Error 1")
	}

	check2 := func(s *terraform.State) error {
		return errors.New("Error 2")
	}

	f := ComposeAggregateTestCheckFunc(check1, check2)
	err := f(nil)
	if err == nil {
		t.Fatalf("Expected errors")
	}

	multi := err.(*multierror.Error)
	if !strings.Contains(multi.Errors[0].Error(), "Error 1") {
		t.Fatalf("Expected Error 1, Got %s", multi.Errors[0])
	}
	if !strings.Contains(multi.Errors[1].Error(), "Error 2") {
		t.Fatalf("Expected Error 2, Got %s", multi.Errors[1])
	}
}

func TestComposeTestCheckFunc(t *testing.T) {
	cases := []struct {
		F      []TestCheckFunc
		Result string
	}{
		{
			F: []TestCheckFunc{
				func(*terraform.State) error { return nil },
			},
			Result: "",
		},

		{
			F: []TestCheckFunc{
				func(*terraform.State) error {
					return fmt.Errorf("error")
				},
				func(*terraform.State) error { return nil },
			},
			Result: "Check 1/2 error: error",
		},

		{
			F: []TestCheckFunc{
				func(*terraform.State) error { return nil },
				func(*terraform.State) error {
					return fmt.Errorf("error")
				},
			},
			Result: "Check 2/2 error: error",
		},

		{
			F: []TestCheckFunc{
				func(*terraform.State) error { return nil },
				func(*terraform.State) error { return nil },
			},
			Result: "",
		},
	}

	for i, tc := range cases {
		f := ComposeTestCheckFunc(tc.F...)
		err := f(nil)
		if err == nil {
			err = fmt.Errorf("")
		}
		if tc.Result != err.Error() {
			t.Fatalf("Case %d bad: %s", i, err)
		}
	}
}

// mockT implements TestT for testing
type mockT struct {
	ErrorCalled    bool
	ErrorArgs      []interface{}
	FatalCalled    bool
	FatalArgs      []interface{}
	ParallelCalled bool
	SkipCalled     bool
	SkipArgs       []interface{}

	f bool
}

func (t *mockT) Error(args ...interface{}) {
	t.ErrorCalled = true
	t.ErrorArgs = args
	t.f = true
}

func (t *mockT) Fatal(args ...interface{}) {
	t.FatalCalled = true
	t.FatalArgs = args
	t.f = true
}

func (t *mockT) Parallel() {
	t.ParallelCalled = true
}

func (t *mockT) Skip(args ...interface{}) {
	t.SkipCalled = true
	t.SkipArgs = args
	t.f = true
}

func (t *mockT) Name() string {
	return "MockedName"
}

func (t *mockT) failed() bool {
	return t.f
}

func (t *mockT) failMessage() string {
	if t.FatalCalled {
		return t.FatalArgs[0].(string)
	} else if t.ErrorCalled {
		return t.ErrorArgs[0].(string)
	} else if t.SkipCalled {
		return t.SkipArgs[0].(string)
	}

	return "unknown"
}

func testProvider() *terraform.MockResourceProvider {
	mp := new(terraform.MockResourceProvider)
	mp.DiffReturn = &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": &terraform.ResourceAttrDiff{
				New: "bar",
			},
		},
	}
	mp.ResourcesReturn = []terraform.ResourceType{
		terraform.ResourceType{Name: "test_instance"},
	}

	return mp
}

func TestTest_Main(t *testing.T) {
	flag.Parse()
	if *flagSweep == "" {
		// Tests for the TestMain method used for Sweepers will panic without the -sweep
		// flag specified. Mock the value for now
		*flagSweep = "us-east-1"
	}

	cases := []struct {
		Name            string
		Sweepers        map[string]*Sweeper
		ExpectedRunList []string
		SweepRun        string
	}{
		{
			Name: "normal",
			Sweepers: map[string]*Sweeper{
				"aws_dummy": &Sweeper{
					Name: "aws_dummy",
					F:    mockSweeperFunc,
				},
			},
			ExpectedRunList: []string{"aws_dummy"},
		},
		{
			Name: "with dep",
			Sweepers: map[string]*Sweeper{
				"aws_dummy": &Sweeper{
					Name: "aws_dummy",
					F:    mockSweeperFunc,
				},
				"aws_top": &Sweeper{
					Name:         "aws_top",
					Dependencies: []string{"aws_sub"},
					F:            mockSweeperFunc,
				},
				"aws_sub": &Sweeper{
					Name: "aws_sub",
					F:    mockSweeperFunc,
				},
			},
			ExpectedRunList: []string{"aws_dummy", "aws_sub", "aws_top"},
		},
		{
			Name: "with filter",
			Sweepers: map[string]*Sweeper{
				"aws_dummy": &Sweeper{
					Name: "aws_dummy",
					F:    mockSweeperFunc,
				},
				"aws_top": &Sweeper{
					Name:         "aws_top",
					Dependencies: []string{"aws_sub"},
					F:            mockSweeperFunc,
				},
				"aws_sub": &Sweeper{
					Name: "aws_sub",
					F:    mockSweeperFunc,
				},
			},
			ExpectedRunList: []string{"aws_dummy"},
			SweepRun:        "aws_dummy",
		},
		{
			Name: "with two filters",
			Sweepers: map[string]*Sweeper{
				"aws_dummy": &Sweeper{
					Name: "aws_dummy",
					F:    mockSweeperFunc,
				},
				"aws_top": &Sweeper{
					Name:         "aws_top",
					Dependencies: []string{"aws_sub"},
					F:            mockSweeperFunc,
				},
				"aws_sub": &Sweeper{
					Name: "aws_sub",
					F:    mockSweeperFunc,
				},
			},
			ExpectedRunList: []string{"aws_dummy", "aws_sub"},
			SweepRun:        "aws_dummy,aws_sub",
		},
		{
			Name: "with dep and filter",
			Sweepers: map[string]*Sweeper{
				"aws_dummy": &Sweeper{
					Name: "aws_dummy",
					F:    mockSweeperFunc,
				},
				"aws_top": &Sweeper{
					Name:         "aws_top",
					Dependencies: []string{"aws_sub"},
					F:            mockSweeperFunc,
				},
				"aws_sub": &Sweeper{
					Name: "aws_sub",
					F:    mockSweeperFunc,
				},
			},
			ExpectedRunList: []string{"aws_top", "aws_sub"},
			SweepRun:        "aws_top",
		},
		{
			Name: "filter and none",
			Sweepers: map[string]*Sweeper{
				"aws_dummy": &Sweeper{
					Name: "aws_dummy",
					F:    mockSweeperFunc,
				},
				"aws_top": &Sweeper{
					Name:         "aws_top",
					Dependencies: []string{"aws_sub"},
					F:            mockSweeperFunc,
				},
				"aws_sub": &Sweeper{
					Name: "aws_sub",
					F:    mockSweeperFunc,
				},
			},
			SweepRun: "none",
		},
	}

	for _, tc := range cases {
		// reset sweepers
		sweeperFuncs = map[string]*Sweeper{}

		t.Run(tc.Name, func(t *testing.T) {
			for n, s := range tc.Sweepers {
				AddTestSweepers(n, s)
			}
			*flagSweepRun = tc.SweepRun

			TestMain(&testing.M{})

			// get list of tests ran from sweeperRunList keys
			var keys []string
			for k, _ := range sweeperRunList {
				keys = append(keys, k)
			}

			sort.Strings(keys)
			sort.Strings(tc.ExpectedRunList)
			if !reflect.DeepEqual(keys, tc.ExpectedRunList) {
				t.Fatalf("Expected keys mismatch, expected:\n%#v\ngot:\n%#v\n", tc.ExpectedRunList, keys)
			}
		})
	}
}

func mockSweeperFunc(s string) error {
	return nil
}

func TestTest_Taint(t *testing.T) {
	t.Skip("test requires new provider implementation")

	mp := testProvider()
	mp.DiffFn = func(
		_ *terraform.InstanceInfo,
		state *terraform.InstanceState,
		_ *terraform.ResourceConfig,
	) (*terraform.InstanceDiff, error) {
		return &terraform.InstanceDiff{
			DestroyTainted: state.Tainted,
		}, nil
	}

	mp.ApplyFn = func(
		info *terraform.InstanceInfo,
		state *terraform.InstanceState,
		diff *terraform.InstanceDiff,
	) (*terraform.InstanceState, error) {
		var id string
		switch {
		case diff.Destroy && !diff.DestroyTainted:
			return nil, nil
		case diff.DestroyTainted:
			id = "tainted"
		default:
			id = "not_tainted"
		}

		return &terraform.InstanceState{
			ID: id,
		}, nil
	}

	mp.RefreshFn = func(
		_ *terraform.InstanceInfo,
		state *terraform.InstanceState,
	) (*terraform.InstanceState, error) {
		return state, nil
	}

	mt := new(mockT)
	Test(mt, TestCase{
		Providers: map[string]terraform.ResourceProvider{
			"test": mp,
		},
		Steps: []TestStep{
			TestStep{
				Config: testConfigStr,
				Check: func(s *terraform.State) error {
					rs := s.RootModule().Resources["test_instance.foo"]
					if rs.Primary.ID != "not_tainted" {
						return fmt.Errorf("expected not_tainted, got %s", rs.Primary.ID)
					}
					return nil
				},
			},
			TestStep{
				Taint:  []string{"test_instance.foo"},
				Config: testConfigStr,
				Check: func(s *terraform.State) error {
					rs := s.RootModule().Resources["test_instance.foo"]
					if rs.Primary.ID != "tainted" {
						return fmt.Errorf("expected tainted, got %s", rs.Primary.ID)
					}
					return nil
				},
			},
			TestStep{
				Taint:       []string{"test_instance.fooo"},
				Config:      testConfigStr,
				ExpectError: regexp.MustCompile("resource \"test_instance.fooo\" not found in state"),
			},
		},
	})

	if mt.failed() {
		t.Fatalf("test failure: %s", mt.failMessage())
	}
}

func TestTestProviderResolver(t *testing.T) {
	stubProvider := func(name string) terraform.ResourceProvider {
		return &schema.Provider{
			Schema: map[string]*schema.Schema{
				name: &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},
			},
		}
	}

	c := TestCase{
		ProviderFactories: map[string]terraform.ResourceProviderFactory{
			"foo": terraform.ResourceProviderFactoryFixed(stubProvider("foo")),
			"bar": terraform.ResourceProviderFactoryFixed(stubProvider("bar")),
		},
		Providers: map[string]terraform.ResourceProvider{
			"baz": stubProvider("baz"),
			"bop": stubProvider("bop"),
		},
	}

	resolver, err := testProviderResolver(c)
	if err != nil {
		t.Fatal(err)
	}

	reqd := discovery.PluginRequirements{
		"foo": &discovery.PluginConstraints{},
		"bar": &discovery.PluginConstraints{},
		"baz": &discovery.PluginConstraints{},
		"bop": &discovery.PluginConstraints{},
	}

	factories, errs := resolver.ResolveProviders(reqd)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Error(err)
		}
		t.Fatal("unexpected errors")
	}

	for name := range reqd {
		t.Run(name, func(t *testing.T) {
			pf, ok := factories[name]
			if !ok {
				t.Fatalf("no factory for %q", name)
			}
			p, err := pf()
			if err != nil {
				t.Fatal(err)
			}
			resp := p.GetSchema()
			_, ok = resp.Provider.Block.Attributes[name]
			if !ok {
				var has string
				for k := range resp.Provider.Block.Attributes {
					has = k
					break
				}
				if has != "" {
					t.Errorf("provider %q does not have the expected schema attribute %q (but has %q)", name, name, has)
				} else {
					t.Errorf("provider %q does not have the expected schema attribute %q", name, name)
				}
			}
		})
	}
}

const testConfigStr = `
resource "test_instance" "foo" {}
`

const testConfigStrProvider = `
provider "test" {}
`

func TestCheckResourceAttr_empty(t *testing.T) {
	s := terraform.NewState()
	s.AddModuleState(&terraform.ModuleState{
		Path: []string{"root"},
		Resources: map[string]*terraform.ResourceState{
			"test_resource": &terraform.ResourceState{
				Primary: &terraform.InstanceState{
					Attributes: map[string]string{
						"empty_list.#": "0",
						"empty_map.%":  "0",
					},
				},
			},
		},
	})

	for _, key := range []string{
		"empty_list.#",
		"empty_map.%",
		"missing_list.#",
		"missing_map.%",
	} {
		t.Run(key, func(t *testing.T) {
			check := TestCheckResourceAttr("test_resource", key, "0")
			if err := check(s); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCheckNoResourceAttr_empty(t *testing.T) {
	s := terraform.NewState()
	s.AddModuleState(&terraform.ModuleState{
		Path: []string{"root"},
		Resources: map[string]*terraform.ResourceState{
			"test_resource": &terraform.ResourceState{
				Primary: &terraform.InstanceState{
					Attributes: map[string]string{
						"empty_list.#": "0",
						"empty_map.%":  "0",
					},
				},
			},
		},
	})

	for _, key := range []string{
		"empty_list.#",
		"empty_map.%",
		"missing_list.#",
		"missing_map.%",
	} {
		t.Run(key, func(t *testing.T) {
			check := TestCheckNoResourceAttr("test_resource", key)
			if err := check(s); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestTestCheckResourceAttrPair(t *testing.T) {
	tests := map[string]struct {
		state   *terraform.State
		wantErr string
	}{
		"exist match": {
			&terraform.State{
				Modules: []*terraform.ModuleState{
					{
						Path: []string{"root"},
						Resources: map[string]*terraform.ResourceState{
							"test.a": {
								Primary: &terraform.InstanceState{
									Attributes: map[string]string{
										"a": "boop",
									},
								},
							},
							"test.b": {
								Primary: &terraform.InstanceState{
									Attributes: map[string]string{
										"b": "boop",
									},
								},
							},
						},
					},
				},
			},
			``,
		},
		"nonexist match": {
			&terraform.State{
				Modules: []*terraform.ModuleState{
					{
						Path: []string{"root"},
						Resources: map[string]*terraform.ResourceState{
							"test.a": {
								Primary: &terraform.InstanceState{
									Attributes: map[string]string{},
								},
							},
							"test.b": {
								Primary: &terraform.InstanceState{
									Attributes: map[string]string{},
								},
							},
						},
					},
				},
			},
			``,
		},
		"exist nonmatch": {
			&terraform.State{
				Modules: []*terraform.ModuleState{
					{
						Path: []string{"root"},
						Resources: map[string]*terraform.ResourceState{
							"test.a": {
								Primary: &terraform.InstanceState{
									Attributes: map[string]string{
										"a": "beep",
									},
								},
							},
							"test.b": {
								Primary: &terraform.InstanceState{
									Attributes: map[string]string{
										"b": "boop",
									},
								},
							},
						},
					},
				},
			},
			`test.a: Attribute 'a' expected "boop", got "beep"`,
		},
		"inconsistent exist a": {
			&terraform.State{
				Modules: []*terraform.ModuleState{
					{
						Path: []string{"root"},
						Resources: map[string]*terraform.ResourceState{
							"test.a": {
								Primary: &terraform.InstanceState{
									Attributes: map[string]string{
										"a": "beep",
									},
								},
							},
							"test.b": {
								Primary: &terraform.InstanceState{
									Attributes: map[string]string{},
								},
							},
						},
					},
				},
			},
			`test.a: Attribute "a" is "beep", but "b" is not set in test.b`,
		},
		"inconsistent exist b": {
			&terraform.State{
				Modules: []*terraform.ModuleState{
					{
						Path: []string{"root"},
						Resources: map[string]*terraform.ResourceState{
							"test.a": {
								Primary: &terraform.InstanceState{
									Attributes: map[string]string{},
								},
							},
							"test.b": {
								Primary: &terraform.InstanceState{
									Attributes: map[string]string{
										"b": "boop",
									},
								},
							},
						},
					},
				},
			},
			`test.a: Attribute "a" not set, but "b" is set in test.b as "boop"`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fn := TestCheckResourceAttrPair("test.a", "a", "test.b", "b")
			err := fn(test.state)

			if test.wantErr != "" {
				if err == nil {
					t.Fatalf("succeeded; want error\nwant: %s", test.wantErr)
				}
				if got, want := err.Error(), test.wantErr; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				return
			}

			if err != nil {
				t.Fatalf("failed; want success\ngot: %s", err.Error())
			}
		})
	}
}

func TestTestCheckResourceAttrPairCount(t *testing.T) {
	tests := map[string]struct {
		state   *terraform.State
		attr    string
		wantErr string
	}{
		"unset and 0 equal list": {
			&terraform.State{
				Modules: []*terraform.ModuleState{
					{
						Path: []string{"root"},
						Resources: map[string]*terraform.ResourceState{
							"test.a": {
								Primary: &terraform.InstanceState{
									Attributes: map[string]string{
										"a.#": "0",
									},
								},
							},
							"test.b": {
								Primary: &terraform.InstanceState{
									Attributes: map[string]string{},
								},
							},
						},
					},
				},
			},
			"a.#",
			``,
		},
		"unset and 0 equal map": {
			&terraform.State{
				Modules: []*terraform.ModuleState{
					{
						Path: []string{"root"},
						Resources: map[string]*terraform.ResourceState{
							"test.a": {
								Primary: &terraform.InstanceState{
									Attributes: map[string]string{
										"a.%": "0",
									},
								},
							},
							"test.b": {
								Primary: &terraform.InstanceState{
									Attributes: map[string]string{},
								},
							},
						},
					},
				},
			},
			"a.%",
			``,
		},
		"count equal": {
			&terraform.State{
				Modules: []*terraform.ModuleState{
					{
						Path: []string{"root"},
						Resources: map[string]*terraform.ResourceState{
							"test.a": {
								Primary: &terraform.InstanceState{
									Attributes: map[string]string{
										"a.%": "1",
									},
								},
							},
							"test.b": {
								Primary: &terraform.InstanceState{
									Attributes: map[string]string{
										"a.%": "1",
									}},
							},
						},
					},
				},
			},
			"a.%",
			``,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fn := TestCheckResourceAttrPair("test.a", test.attr, "test.b", test.attr)
			err := fn(test.state)

			if test.wantErr != "" {
				if err == nil {
					t.Fatalf("succeeded; want error\nwant: %s", test.wantErr)
				}
				if got, want := err.Error(), test.wantErr; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				return
			}

			if err != nil {
				t.Fatalf("failed; want success\ngot: %s", err.Error())
			}
		})
	}
}
