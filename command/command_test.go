package command

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

// This is the directory where our test fixtures are.
var fixtureDir = "./test-fixtures"

func init() {
	// Expand the fixture dir on init because we change the working
	// directory in some tests.
	var err error
	fixtureDir, err = filepath.Abs(fixtureDir)
	if err != nil {
		panic(err)
	}
}

func testFixturePath(name string) string {
	return filepath.Join(fixtureDir, name)
}

func testCtxConfig(p terraform.ResourceProvider) *terraform.ContextOpts {
	return &terraform.ContextOpts{
		Providers: map[string]terraform.ResourceProviderFactory{
			"test": func() (terraform.ResourceProvider, error) {
				return p, nil
			},
		},
	}
}

func testPlanFile(t *testing.T, plan *terraform.Plan) string {
	path := testTempFile(t)

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	if err := terraform.WritePlan(plan, f); err != nil {
		t.Fatalf("err: %s", err)
	}

	return path
}

func testReadPlan(t *testing.T, path string) *terraform.Plan {
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	p, err := terraform.ReadPlan(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return p
}

func testStateFile(t *testing.T, s *terraform.State) string {
	path := testTempFile(t)

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	if err := terraform.WriteState(s, f); err != nil {
		t.Fatalf("err: %s", err)
	}

	return path
}

func testProvider() *terraform.MockResourceProvider {
	p := new(terraform.MockResourceProvider)
	p.DiffReturn = &terraform.ResourceDiff{}
	p.RefreshFn = func(
		s *terraform.ResourceState) (*terraform.ResourceState, error) {
		return s, nil
	}
	p.ResourcesReturn = []terraform.ResourceType{
		terraform.ResourceType{
			Name: "test_instance",
		},
	}

	return p
}

func testTempFile(t *testing.T) string {
	tf, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	result := tf.Name()

	if err := tf.Close(); err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Remove(result); err != nil {
		t.Fatalf("err: %s", err)
	}

	return result
}

func testTempDir(t *testing.T) string {
	d, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return d
}
