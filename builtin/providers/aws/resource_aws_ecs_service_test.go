package aws

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestParseTaskDefinition(t *testing.T) {
	cases := map[string]map[string]interface{}{
		"invalid": map[string]interface{}{
			"family":   "",
			"revision": "",
			"isValid":  false,
		},
		"invalidWithColon:": map[string]interface{}{
			"family":   "",
			"revision": "",
			"isValid":  false,
		},
		"1234": map[string]interface{}{
			"family":   "",
			"revision": "",
			"isValid":  false,
		},
		"invalid:aaa": map[string]interface{}{
			"family":   "",
			"revision": "",
			"isValid":  false,
		},
		"invalid=family:1": map[string]interface{}{
			"family":   "",
			"revision": "",
			"isValid":  false,
		},
		"invalid:name:1": map[string]interface{}{
			"family":   "",
			"revision": "",
			"isValid":  false,
		},
		"valid:1": map[string]interface{}{
			"family":   "valid",
			"revision": "1",
			"isValid":  true,
		},
		"abc12-def:54": map[string]interface{}{
			"family":   "abc12-def",
			"revision": "54",
			"isValid":  true,
		},
		"lorem_ip-sum:123": map[string]interface{}{
			"family":   "lorem_ip-sum",
			"revision": "123",
			"isValid":  true,
		},
		"lorem-ipsum:1": map[string]interface{}{
			"family":   "lorem-ipsum",
			"revision": "1",
			"isValid":  true,
		},
	}

	for input, expectedOutput := range cases {
		family, revision, err := parseTaskDefinition(input)
		isValid := expectedOutput["isValid"].(bool)
		if !isValid && err == nil {
			t.Fatalf("Task definition %s should fail", input)
		}

		expectedFamily := expectedOutput["family"].(string)
		if family != expectedFamily {
			t.Fatalf("Unexpected family (%#v) for task definition %s\n%#v", family, input, err)
		}
		expectedRevision := expectedOutput["revision"].(string)
		if revision != expectedRevision {
			t.Fatalf("Unexpected revision (%#v) for task definition %s\n%#v", revision, input, err)
		}
	}
}

func TestAccAWSEcsServiceWithARN(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEcsService,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.mongo"),
				),
			},

			resource.TestStep{
				Config: testAccAWSEcsServiceModified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.mongo"),
				),
			},
		},
	})
}

func TestAccAWSEcsServiceWithFamilyAndRevision(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEcsServiceWithFamilyAndRevision,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.jenkins"),
				),
			},

			resource.TestStep{
				Config: testAccAWSEcsServiceWithFamilyAndRevisionModified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.jenkins"),
				),
			},
		},
	})
}

// Regression for https://github.com/hashicorp/terraform/issues/2427
func TestAccAWSEcsServiceWithRenamedCluster(t *testing.T) {
	originalRegexp := regexp.MustCompile(
		"^arn:aws:ecs:[^:]+:[0-9]+:cluster/terraformecstest3$")
	modifiedRegexp := regexp.MustCompile(
		"^arn:aws:ecs:[^:]+:[0-9]+:cluster/terraformecstest3modified$")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEcsServiceWithRenamedCluster,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.ghost"),
					resource.TestMatchResourceAttr(
						"aws_ecs_service.ghost", "cluster", originalRegexp),
				),
			},

			resource.TestStep{
				Config: testAccAWSEcsServiceWithRenamedClusterModified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.ghost"),
					resource.TestMatchResourceAttr(
						"aws_ecs_service.ghost", "cluster", modifiedRegexp),
				),
			},
		},
	})
}

func testAccCheckAWSEcsServiceDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ecsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ecs_service" {
			continue
		}

		out, err := conn.DescribeServices(&ecs.DescribeServicesInput{
			Services: []*string{aws.String(rs.Primary.ID)},
		})

		if err == nil {
			if len(out.Services) > 0 {
				return fmt.Errorf("ECS service still exists:\n%#v", out.Services)
			}
		}

		return err
	}

	return nil
}

func testAccCheckAWSEcsServiceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		return nil
	}
}

var testAccAWSEcsService = `
resource "aws_ecs_cluster" "default" {
	name = "terraformecstest1"
}

resource "aws_ecs_task_definition" "mongo" {
  family = "mongodb"
  container_definitions = <<DEFINITION
[
  {
    "cpu": 128,
    "essential": true,
    "image": "mongo:latest",
    "memory": 128,
    "name": "mongodb"
  }
]
DEFINITION
}

resource "aws_ecs_service" "mongo" {
  name = "mongodb"
  cluster = "${aws_ecs_cluster.default.id}"
  task_definition = "${aws_ecs_task_definition.mongo.arn}"
  desired_count = 1
}
`

var testAccAWSEcsServiceModified = `
resource "aws_ecs_cluster" "default" {
	name = "terraformecstest1"
}

resource "aws_ecs_task_definition" "mongo" {
  family = "mongodb"
  container_definitions = <<DEFINITION
[
  {
    "cpu": 128,
    "essential": true,
    "image": "mongo:latest",
    "memory": 128,
    "name": "mongodb"
  }
]
DEFINITION
}

resource "aws_ecs_service" "mongo" {
  name = "mongodb"
  cluster = "${aws_ecs_cluster.default.id}"
  task_definition = "${aws_ecs_task_definition.mongo.arn}"
  desired_count = 2
}
`

var testAccAWSEcsServiceWithFamilyAndRevision = `
resource "aws_ecs_cluster" "default" {
	name = "terraformecstest2"
}

resource "aws_ecs_task_definition" "jenkins" {
  family = "jenkins"
  container_definitions = <<DEFINITION
[
  {
    "cpu": 128,
    "essential": true,
    "image": "jenkins:latest",
    "memory": 128,
    "name": "jenkins"
  }
]
DEFINITION
}

resource "aws_ecs_service" "jenkins" {
  name = "jenkins"
  cluster = "${aws_ecs_cluster.default.id}"
  task_definition = "${aws_ecs_task_definition.jenkins.family}:${aws_ecs_task_definition.jenkins.revision}"
  desired_count = 1
}
`

var testAccAWSEcsServiceWithFamilyAndRevisionModified = `
resource "aws_ecs_cluster" "default" {
	name = "terraformecstest2"
}

resource "aws_ecs_task_definition" "jenkins" {
  family = "jenkins"
  container_definitions = <<DEFINITION
[
  {
    "cpu": 128,
    "essential": true,
    "image": "jenkins:latest",
    "memory": 128,
    "name": "jenkins"
  }
]
DEFINITION
}

resource "aws_ecs_service" "jenkins" {
  name = "jenkins"
  cluster = "${aws_ecs_cluster.default.id}"
  task_definition = "${aws_ecs_task_definition.jenkins.family}:${aws_ecs_task_definition.jenkins.revision}"
  desired_count = 1
}
`

var testAccAWSEcsServiceWithRenamedCluster = `
resource "aws_ecs_cluster" "default" {
	name = "terraformecstest3"
}
resource "aws_ecs_task_definition" "ghost" {
  family = "ghost"
  container_definitions = <<DEFINITION
[
  {
    "cpu": 128,
    "essential": true,
    "image": "ghost:latest",
    "memory": 128,
    "name": "ghost"
  }
]
DEFINITION
}
resource "aws_ecs_service" "ghost" {
  name = "ghost"
  cluster = "${aws_ecs_cluster.default.id}"
  task_definition = "${aws_ecs_task_definition.ghost.family}:${aws_ecs_task_definition.ghost.revision}"
  desired_count = 1
}
`

var testAccAWSEcsServiceWithRenamedClusterModified = `
resource "aws_ecs_cluster" "default" {
	name = "terraformecstest3modified"
}
resource "aws_ecs_task_definition" "ghost" {
  family = "ghost"
  container_definitions = <<DEFINITION
[
  {
    "cpu": 128,
    "essential": true,
    "image": "ghost:latest",
    "memory": 128,
    "name": "ghost"
  }
]
DEFINITION
}
resource "aws_ecs_service" "ghost" {
  name = "ghost"
  cluster = "${aws_ecs_cluster.default.id}"
  task_definition = "${aws_ecs_task_definition.ghost.family}:${aws_ecs_task_definition.ghost.revision}"
  desired_count = 1
}
`
