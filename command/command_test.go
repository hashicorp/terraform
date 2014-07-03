package command

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./test-fixtures"

func testFixturePath(name string) string {
	return filepath.Join(fixtureDir, name, "main.tf")
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
