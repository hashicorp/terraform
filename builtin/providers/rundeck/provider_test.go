package rundeck

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// To run these acceptance tests, you will need a Rundeck server.
// An easy way to get one is to use Rundeck's "Anvils" demo, which includes a Vagrantfile
// to get it running easily:
//    https://github.com/rundeck/anvils-demo
// The anvils demo ships with some example security policies that don't have enough access to
// run the tests, so you need to either modify one of the stock users to have full access or
// create a new user with such access. The following block is an example that gives the
// 'admin' user and API clients open access.
// In the anvils demo the admin password is "admin" by default.

// Place the contents of the following comment in /etc/rundeck/terraform-test.aclpolicy
/*
description: Admin, all access.
context:
  project: '.*' # all projects
for:
  resource:
    - allow: '*' # allow read/create all kinds
  adhoc:
    - allow: '*' # allow read/running/killing adhoc jobs
  job:
    - allow: '*' # allow read/write/delete/run/kill of all jobs
  node:
    - allow: '*' # allow read/run for all nodes
by:
  group: admin
---
description: Admin, all access.
context:
  application: 'rundeck'
for:
  resource:
    - allow: '*' # allow create of projects
  project:
    - allow: '*' # allow view/admin of all projects
  storage:
    - allow: '*' # allow read/create/update/delete for all /keys/* storage content
by:
  group: admin
---
description: Admin API, all access.
context:
  application: 'rundeck'
for:
  resource:
    - allow: '*' # allow create of projects
  project:
    - allow: '*' # allow view/admin of all projects
  storage:
    - allow: '*' # allow read/create/update/delete for all /keys/* storage content
by:
  group: api_token_group
*/

// Once you've got a user set up, put that user's API auth token in the RUNDECK_AUTH_TOKEN
// environment variable, and put the URL of the Rundeck home page in the RUNDECK_URL variable.
// If you're using the Anvils demo in its default configuration, you can find or generate an API
// token at http://192.168.50.2:4440/user/profile once you've logged in, and RUNDECK_URL will
// be http://192.168.50.2:4440/ .

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"rundeck": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("RUNDECK_URL"); v == "" {
		t.Fatal("RUNDECK_URL must be set for acceptance tests")
	}
	if v := os.Getenv("RUNDECK_AUTH_TOKEN"); v == "" {
		t.Fatal("RUNDECK_AUTH_TOKEN must be set for acceptance tests")
	}
}
