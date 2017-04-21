package rancher

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	rancherClient "github.com/rancher/go-rancher/client"
)

func TestAccRancherStack_basic(t *testing.T) {
	var stack rancherClient.Environment

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRancherStackDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRancherStackConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRancherStackExists("rancher_stack.foo", &stack),
					resource.TestCheckResourceAttr("rancher_stack.foo", "name", "foo"),
					resource.TestCheckResourceAttr("rancher_stack.foo", "description", "Terraform acc test group"),
					resource.TestCheckResourceAttr("rancher_stack.foo", "catalog_id", ""),
					resource.TestCheckResourceAttr("rancher_stack.foo", "docker_compose", ""),
					resource.TestCheckResourceAttr("rancher_stack.foo", "rancher_compose", ""),
					testAccCheckRancherStackAttributes(&stack, emptyEnvironment, false),
				),
			},
			resource.TestStep{
				Config: testAccRancherStackUpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRancherStackExists("rancher_stack.foo", &stack),
					resource.TestCheckResourceAttr("rancher_stack.foo", "name", "foo2"),
					resource.TestCheckResourceAttr("rancher_stack.foo", "description", "Terraform acc test group - updated"),
					resource.TestCheckResourceAttr("rancher_stack.foo", "catalog_id", ""),
					resource.TestCheckResourceAttr("rancher_stack.foo", "docker_compose", ""),
					resource.TestCheckResourceAttr("rancher_stack.foo", "rancher_compose", ""),
					testAccCheckRancherStackAttributes(&stack, emptyEnvironment, false),
				),
			},
		},
	})
}

func TestAccRancherStack_compose(t *testing.T) {
	var stack rancherClient.Environment

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRancherStackDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRancherStackComposeConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRancherStackExists("rancher_stack.compose", &stack),
					resource.TestCheckResourceAttr("rancher_stack.compose", "name", "compose"),
					resource.TestCheckResourceAttr("rancher_stack.compose", "description", "Terraform acc test group - compose"),
					resource.TestCheckResourceAttr("rancher_stack.compose", "catalog_id", ""),
					resource.TestCheckResourceAttr("rancher_stack.compose", "docker_compose", "web: { image: nginx }"),
					resource.TestCheckResourceAttr("rancher_stack.compose", "rancher_compose", "web: { scale: 1 }"),
					testAccCheckRancherStackAttributes(&stack, emptyEnvironment, false),
				),
			},
		},
	})
}

//The following tests are run against the Default environment because
//upgrading a stack automatically starts the services which never
//completes if there is no host available
func TestAccRancherStack_catalog(t *testing.T) {
	var stack rancherClient.Environment

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRancherStackDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRancherStackSystemCatalogConfigInitial,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRancherStackExists("rancher_stack.catalog", &stack),
					resource.TestCheckResourceAttr("rancher_stack.catalog", "name", "catalogInitial"),
					resource.TestCheckResourceAttr("rancher_stack.catalog", "description", "Terraform acc test group - catalogInitial"),
					resource.TestCheckResourceAttr("rancher_stack.catalog", "catalog_id", "community:janitor:0"),
					resource.TestCheckResourceAttr("rancher_stack.catalog", "scope", "system"),
					resource.TestCheckResourceAttr("rancher_stack.catalog", "docker_compose", ""),
					resource.TestCheckResourceAttr("rancher_stack.catalog", "rancher_compose", ""),
					resource.TestCheckResourceAttr("rancher_stack.catalog", "rendered_docker_compose", catalogDockerComposeInitial),
					resource.TestCheckResourceAttr("rancher_stack.catalog", "rendered_rancher_compose", catalogRancherComposeInitial),
					testAccCheckRancherStackAttributes(&stack, catalogEnvironment, true),
				),
			},
			resource.TestStep{
				Config: testAccRancherStackSystemCatalogConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRancherStackExists("rancher_stack.catalog", &stack),
					resource.TestCheckResourceAttr("rancher_stack.catalog", "name", "catalogUpdate"),
					resource.TestCheckResourceAttr("rancher_stack.catalog", "description", "Terraform acc test group - catalogUpdate"),
					resource.TestCheckResourceAttr("rancher_stack.catalog", "catalog_id", "community:janitor:1"),
					resource.TestCheckResourceAttr("rancher_stack.catalog", "scope", "user"),
					resource.TestCheckResourceAttr("rancher_stack.catalog", "docker_compose", ""),
					resource.TestCheckResourceAttr("rancher_stack.catalog", "rancher_compose", ""),
					resource.TestCheckResourceAttr("rancher_stack.catalog", "rendered_docker_compose", catalogDockerComposeUpdate),
					resource.TestCheckResourceAttr("rancher_stack.catalog", "rendered_rancher_compose", catalogRancherComposeUpdate),
					testAccCheckRancherStackAttributes(&stack, catalogEnvironmentUpgrade, true),
				),
			},
		},
	})
}

func TestAccRancherStack_disappears(t *testing.T) {
	var stack rancherClient.Environment

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRancherStackDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRancherStackConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRancherStackExists("rancher_stack.foo", &stack),
					testAccRancherStackDisappears(&stack),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccRancherStackDisappears(stack *rancherClient.Environment) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := testAccProvider.Meta().(*Config).EnvironmentClient(stack.AccountId)
		if err != nil {
			return err
		}

		if err := client.Environment.Delete(stack); err != nil {
			return fmt.Errorf("Error deleting Stack: %s", err)
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"active", "removed", "removing"},
			Target:     []string{"removed"},
			Refresh:    StackStateRefreshFunc(client, stack.Id),
			Timeout:    10 * time.Minute,
			Delay:      1 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, waitErr := stateConf.WaitForState()
		if waitErr != nil {
			return fmt.Errorf(
				"Error waiting for stack (%s) to be removed: %s", stack.Id, waitErr)
		}

		return nil
	}
}

func testAccCheckRancherStackExists(n string, stack *rancherClient.Environment) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No App Name is set")
		}

		client, err := testAccProvider.Meta().(*Config).EnvironmentClient(rs.Primary.Attributes["environment_id"])
		if err != nil {
			return err
		}

		foundStack, err := client.Environment.ById(rs.Primary.ID)
		if err != nil {
			return err
		}

		if foundStack.Resource.Id != rs.Primary.ID {
			return fmt.Errorf("Stack not found")
		}

		*stack = *foundStack

		return nil
	}
}

func testAccCheckRancherStackAttributes(stack *rancherClient.Environment, environment map[string]string, startOnCreate bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if len(stack.Environment) != len(environment) {
			return fmt.Errorf("Bad environment size: %v should be: %v", len(stack.Environment), environment)
		}

		for k, v := range stack.Environment {
			if environment[k] != v {
				return fmt.Errorf("Bad environment value for %s: %s should be: %s", k, environment[k], v)
			}
		}

		if stack.StartOnCreate != startOnCreate {
			return fmt.Errorf("Bad startOnCreate: %t should be: %t", stack.StartOnCreate, startOnCreate)
		}

		return nil
	}
}

func testAccCheckRancherStackDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "rancher_stack" {
			continue
		}
		client, err := testAccProvider.Meta().(*Config).GlobalClient()
		if err != nil {
			return err
		}

		stack, err := client.Environment.ById(rs.Primary.ID)

		if err == nil {
			if stack != nil &&
				stack.Resource.Id == rs.Primary.ID &&
				stack.State != "removed" {
				return fmt.Errorf("Stack still exists")
			}
		}

		return nil
	}
	return nil
}

const testAccRancherStackConfig = `
resource "rancher_environment" "foo" {
	name = "foo"
}

resource "rancher_stack" "foo" {
	name = "foo"
	description = "Terraform acc test group"
	environment_id = "${rancher_environment.foo.id}"
}
`

const testAccRancherStackUpdateConfig = `
resource "rancher_environment" "foo" {
	name = "foo"
}

resource "rancher_stack" "foo" {
	name = "foo2"
	description = "Terraform acc test group - updated"
	environment_id = "${rancher_environment.foo.id}"
}
`

const testAccRancherStackComposeConfig = `
resource "rancher_environment" "foo" {
	name = "foo"
}

resource "rancher_stack" "compose" {
	name = "compose"
	description = "Terraform acc test group - compose"
	environment_id = "${rancher_environment.foo.id}"
	docker_compose = "web: { image: nginx }"
	rancher_compose = "web: { scale: 1 }"
}
`

const testAccRancherStackSystemCatalogConfigInitial = `
resource "rancher_stack" "catalog" {
	name = "catalogInitial"
	description = "Terraform acc test group - catalogInitial"
	environment_id = "1a5"
	catalog_id = "community:janitor:0"
	scope = "system"
	start_on_create = true
	environment {
		EXCLUDE_LABEL = "cleanup=false"
		FREQUENCY = "60"
		KEEP = "rancher/agent:*"
	}
}
`

const testAccRancherStackSystemCatalogConfigUpdate = `
resource "rancher_stack" "catalog" {
	name = "catalogUpdate"
	description = "Terraform acc test group - catalogUpdate"
	environment_id = "1a5"
	catalog_id = "community:janitor:1"
	scope = "user"
	environment {
		EXCLUDE_LABEL = "cleanup=false"
		FREQUENCY = "60"
		KEEP = "rancher/agent:*"
		KEEPC = "*:*"
	}
}
`

var catalogDockerComposeInitial = `cleanup:
  environment:
    CLEAN_PERIOD: '60'
    DELAY_TIME: '900'
    KEEP_IMAGES: rancher/agent:*
  labels:
    io.rancher.scheduler.global: 'true'
    io.rancher.scheduler.affinity:host_label_ne: cleanup=false
  tty: true
  image: meltwater/docker-cleanup:1.4.0
  privileged: true
  volumes:
  - /var/run/docker.sock:/var/run/docker.sock
  - /var/lib/docker:/var/lib/docker
  stdin_open: true
`

const catalogRancherComposeInitial = `{}
`

const catalogDockerComposeUpdate = `cleanup:
  environment:
    CLEAN_PERIOD: '60'
    DELAY_TIME: '900'
    KEEP_CONTAINERS: '*:*'
    KEEP_IMAGES: rancher/agent:*
  labels:
    io.rancher.scheduler.global: 'true'
    io.rancher.scheduler.affinity:host_label_ne: cleanup=false
  image: sshipway/docker-cleanup:1.5.2
  volumes:
  - /var/run/docker.sock:/var/run/docker.sock
  - /var/lib/docker:/var/lib/docker
  net: none
`

const catalogRancherComposeUpdate = `{}
`

var emptyEnvironment = map[string]string{}

var catalogEnvironment = map[string]string{
	"EXCLUDE_LABEL": "cleanup=false",
	"FREQUENCY":     "60",
	"KEEP":          "rancher/agent:*",
}

var catalogEnvironmentUpgrade = map[string]string{
	"EXCLUDE_LABEL": "cleanup=false",
	"FREQUENCY":     "60",
	"KEEP":          "rancher/agent:*",
	"KEEPC":         "*:*",
}
