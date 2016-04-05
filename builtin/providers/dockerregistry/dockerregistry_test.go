package dockerregistry

// Note: to run the tests that actually talk to the registry, run:
//
// $ TF_ACC=1 go test -v github.com/hashicorp/terraform/builtin/providers/dockerregistry
//
// (Username and password are not necessary as it only reads a public
// repository. No tests currently test auth.)

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"dockerregistry": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestAccDockerRegistry_good(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerRegistryConfigGood,
				Check:  resource.TestCheckResourceAttr("data.dockerregistry_image.good", "id", "library/alpine:3.1"),
			},
		},
	})
}

func TestAccDockerRegistry_missing_tag(t *testing.T) {
	expectT := &expectOneErrorTestT{
		ErrorPredicate: func(args ...interface{}) bool {
			return len(args) == 1 && strings.Contains(args[0].(string),
				"Docker image library/alpine:3.1-does-not-exist not found in registry")
		},
		WrappedT: t,
	}
	resource.Test(expectT, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerRegistryConfigMissingTag,
			},
		},
	})
	if !expectT.GotError {
		t.Error("Did not get expected error")
	}
}

func TestAccDockerRegistry_missing_repo(t *testing.T) {
	expectT := &expectOneErrorTestT{
		ErrorPredicate: func(args ...interface{}) bool {
			if len(args) != 1 {
				return false
			}
			e := args[0].(string)
			// This is not the best error, but it's what the registry gives us.
			return strings.Contains(e, "Error looking up tags for library/alpine-does-not-exist") &&
				strings.Contains(e, "UNAUTHORIZED")
		},
		WrappedT: t,
	}
	resource.Test(expectT, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerRegistryConfigMissingRepo,
			},
		},
	})
	if !expectT.GotError {
		t.Error("Did not get expected error")
	}
}

const testAccDockerRegistryConfigGood = `
data "dockerregistry_image" "good" {
  repository = "library/alpine"
  tag = "3.1"
}
`

const testAccDockerRegistryConfigMissingTag = `
data "dockerregistry_image" "bad" {
  repository = "library/alpine"
  tag = "3.1-does-not-exist"
}
`

const testAccDockerRegistryConfigMissingRepo = `
data "dockerregistry_image" "bad" {
  repository = "library/alpine-does-not-exist"
  tag = "3.1"
}
`

// This implements the terraform TestT interface and expects to have Error
// called on it exactly once. Skip is passed through.
type expectOneErrorTestT struct {
	ErrorPredicate func(args ...interface{}) bool
	WrappedT       *testing.T
	GotError       bool
}

func (t *expectOneErrorTestT) Fatal(args ...interface{}) {
	t.WrappedT.Fatal(args...)
}

func (t *expectOneErrorTestT) Skip(args ...interface{}) {
	t.WrappedT.Skip(args...)
}

func (t *expectOneErrorTestT) Error(args ...interface{}) {
	if t.GotError {
		t.WrappedT.Error("Got unexpected additional error:", fmt.Sprintln(args...))
		return
	}
	if !t.ErrorPredicate(args...) {
		t.WrappedT.Error("Got non-matching error:", fmt.Sprintln(args...))
		return
	}
	t.GotError = true
}
