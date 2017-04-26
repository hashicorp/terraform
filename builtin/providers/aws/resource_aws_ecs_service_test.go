package aws

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestParseTaskDefinition(t *testing.T) {
	cases := map[string]map[string]interface{}{
		"invalid": {
			"family":   "",
			"revision": "",
			"isValid":  false,
		},
		"invalidWithColon:": {
			"family":   "",
			"revision": "",
			"isValid":  false,
		},
		"1234": {
			"family":   "",
			"revision": "",
			"isValid":  false,
		},
		"invalid:aaa": {
			"family":   "",
			"revision": "",
			"isValid":  false,
		},
		"invalid=family:1": {
			"family":   "",
			"revision": "",
			"isValid":  false,
		},
		"invalid:name:1": {
			"family":   "",
			"revision": "",
			"isValid":  false,
		},
		"valid:1": {
			"family":   "valid",
			"revision": "1",
			"isValid":  true,
		},
		"abc12-def:54": {
			"family":   "abc12-def",
			"revision": "54",
			"isValid":  true,
		},
		"lorem_ip-sum:123": {
			"family":   "lorem_ip-sum",
			"revision": "123",
			"isValid":  true,
		},
		"lorem-ipsum:1": {
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
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsService(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.mongo"),
				),
			},

			{
				Config: testAccAWSEcsServiceModified(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.mongo"),
				),
			},
		},
	})
}

func TestAccAWSEcsServiceWithFamilyAndRevision(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-test")
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsServiceWithFamilyAndRevision(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.jenkins"),
				),
			},

			{
				Config: testAccAWSEcsServiceWithFamilyAndRevisionModified(rName),
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
			{
				Config: testAccAWSEcsServiceWithRenamedCluster,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.ghost"),
					resource.TestMatchResourceAttr(
						"aws_ecs_service.ghost", "cluster", originalRegexp),
				),
			},

			{
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

func TestAccAWSEcsService_withIamRole(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsService_withIamRole,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.ghost"),
				),
			},
		},
	})
}

func TestAccAWSEcsService_withDeploymentValues(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsServiceWithDeploymentValues(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.mongo"),
					resource.TestCheckResourceAttr(
						"aws_ecs_service.mongo", "deployment_maximum_percent", "200"),
					resource.TestCheckResourceAttr(
						"aws_ecs_service.mongo", "deployment_minimum_healthy_percent", "100"),
				),
			},
		},
	})
}

// Regression for https://github.com/hashicorp/terraform/issues/3444
func TestAccAWSEcsService_withLbChanges(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsService_withLbChanges,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.with_lb_changes"),
				),
			},
			{
				Config: testAccAWSEcsService_withLbChanges_modified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.with_lb_changes"),
				),
			},
		},
	})
}

// Regression for https://github.com/hashicorp/terraform/issues/3361
func TestAccAWSEcsService_withEcsClusterName(t *testing.T) {
	clusterName := regexp.MustCompile("^terraformecstestcluster$")
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsServiceWithEcsClusterName,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.jenkins"),
					resource.TestMatchResourceAttr(
						"aws_ecs_service.jenkins", "cluster", clusterName),
				),
			},
		},
	})
}

func TestAccAWSEcsService_withAlb(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsServiceWithAlb(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.with_alb"),
				),
			},
		},
	})
}

func TestAccAWSEcsServiceWithPlacementStrategy(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsService(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.mongo"),
					resource.TestCheckResourceAttr("aws_ecs_service.mongo", "placement_strategy.#", "0"),
				),
			},
			{
				Config: testAccAWSEcsServiceWithPlacementStrategy(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.mongo"),
					resource.TestCheckResourceAttr("aws_ecs_service.mongo", "placement_strategy.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSEcsServiceWithPlacementConstraints(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsServiceWithPlacementConstraint(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.mongo"),
					resource.TestCheckResourceAttr("aws_ecs_service.mongo", "placement_constraints.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSEcsServiceWithPlacementConstraints_emptyExpression(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEcsServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsServiceWithPlacementConstraintEmptyExpression(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsServiceExists("aws_ecs_service.mongo"),
					resource.TestCheckResourceAttr("aws_ecs_service.mongo", "placement_constraints.#", "1"),
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
			Cluster:  aws.String(rs.Primary.Attributes["cluster"]),
		})

		if err == nil {
			if len(out.Services) > 0 {
				var activeServices []*ecs.Service
				for _, svc := range out.Services {
					if *svc.Status != "INACTIVE" {
						activeServices = append(activeServices, svc)
					}
				}
				if len(activeServices) == 0 {
					return nil
				}

				return fmt.Errorf("ECS service still exists:\n%#v", activeServices)
			}
			return nil
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

func testAccAWSEcsService(rInt int) string {
	return fmt.Sprintf(`
resource "aws_ecs_cluster" "default" {
	name = "terraformecstest%d"
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
  name = "mongodb-%d"
  cluster = "${aws_ecs_cluster.default.id}"
  task_definition = "${aws_ecs_task_definition.mongo.arn}"
  desired_count = 1
}
`, rInt, rInt)
}

func testAccAWSEcsServiceModified(rInt int) string {
	return fmt.Sprintf(`
resource "aws_ecs_cluster" "default" {
	name = "terraformecstest%d"
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
  name = "mongodb-%d"
  cluster = "${aws_ecs_cluster.default.id}"
  task_definition = "${aws_ecs_task_definition.mongo.arn}"
  desired_count = 2
}
`, rInt, rInt)
}

func testAccAWSEcsServiceWithPlacementStrategy(rInt int) string {
	return fmt.Sprintf(`
resource "aws_ecs_cluster" "default" {
	name = "terraformecstest%d"
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
  name = "mongodb-%d"
  cluster = "${aws_ecs_cluster.default.id}"
  task_definition = "${aws_ecs_task_definition.mongo.arn}"
  desired_count = 1
  placement_strategy {
	type = "binpack"
	field = "memory"
  }
}
`, rInt, rInt)
}

func testAccAWSEcsServiceWithPlacementConstraint(rInt int) string {
	return fmt.Sprintf(`
	resource "aws_ecs_cluster" "default" {
		name = "terraformecstest%d"
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
	  name = "mongodb-%d"
	  cluster = "${aws_ecs_cluster.default.id}"
	  task_definition = "${aws_ecs_task_definition.mongo.arn}"
	  desired_count = 1
	  placement_constraints {
		type = "memberOf"
		expression = "attribute:ecs.availability-zone in [us-west-2a, us-west-2b]"
	  }
	}
	`, rInt, rInt)
}

func testAccAWSEcsServiceWithPlacementConstraintEmptyExpression(rInt int) string {
	return fmt.Sprintf(`
resource "aws_ecs_cluster" "default" {
	name = "terraformecstest%d"
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
  name = "mongodb-%d"
  cluster = "${aws_ecs_cluster.default.id}"
  task_definition = "${aws_ecs_task_definition.mongo.arn}"
  desired_count = 1
  placement_constraints {
	  type = "distinctInstance"
  }
}
`, rInt, rInt)
}

var testAccAWSEcsService_withIamRole = `
resource "aws_ecs_cluster" "main" {
	name = "terraformecstest11"
}

resource "aws_ecs_task_definition" "ghost" {
  family = "ghost_service"
  container_definitions = <<DEFINITION
[
  {
    "cpu": 128,
    "essential": true,
    "image": "ghost:latest",
    "memory": 128,
    "name": "ghost",
    "portMappings": [
      {
        "containerPort": 2368,
        "hostPort": 8080
      }
    ]
  }
]
DEFINITION
}

resource "aws_iam_role" "ecs_service" {
    name = "EcsService"
    assume_role_policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": "sts:AssumeRole",
            "Principal": {"AWS": "*"},
            "Effect": "Allow",
            "Sid": ""
        }
    ]
}
EOF
}

resource "aws_iam_role_policy" "ecs_service" {
    name = "EcsService"
    role = "${aws_iam_role.ecs_service.name}"
    policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "elasticloadbalancing:*",
        "ec2:*",
        "ecs:*"
      ],
      "Resource": [
        "*"
      ]
    }
  ]
}
EOF
}

resource "aws_elb" "main" {
  availability_zones = ["us-west-2a"]

  listener {
    instance_port = 8080
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }
}

resource "aws_ecs_service" "ghost" {
  name = "ghost"
  cluster = "${aws_ecs_cluster.main.id}"
  task_definition = "${aws_ecs_task_definition.ghost.arn}"
  desired_count = 1
  iam_role = "${aws_iam_role.ecs_service.name}"

  load_balancer {
    elb_name = "${aws_elb.main.id}"
    container_name = "ghost"
    container_port = "2368"
  }

  depends_on = ["aws_iam_role_policy.ecs_service"]
}
`

func testAccAWSEcsServiceWithDeploymentValues(rInt int) string {
	return fmt.Sprintf(`
resource "aws_ecs_cluster" "default" {
	name = "terraformecstest-%d"
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
  name = "mongodb-%d"
  cluster = "${aws_ecs_cluster.default.id}"
  task_definition = "${aws_ecs_task_definition.mongo.arn}"
  desired_count = 1
}
`, rInt, rInt)
}

var tpl_testAccAWSEcsService_withLbChanges = `
resource "aws_ecs_cluster" "main" {
	name = "terraformecstest12"
}

resource "aws_ecs_task_definition" "with_lb_changes" {
  family = "ghost_lbd"
  container_definitions = <<DEFINITION
[
  {
    "cpu": 128,
    "essential": true,
    "image": "%s",
    "memory": 128,
    "name": "%s",
    "portMappings": [
      {
        "containerPort": %d,
        "hostPort": %d
      }
    ]
  }
]
DEFINITION
}

resource "aws_iam_role" "ecs_service" {
    name = "EcsServiceLbd"
    assume_role_policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": "sts:AssumeRole",
            "Principal": {"AWS": "*"},
            "Effect": "Allow",
            "Sid": ""
        }
    ]
}
EOF
}

resource "aws_iam_role_policy" "ecs_service" {
    name = "EcsServiceLbd"
    role = "${aws_iam_role.ecs_service.name}"
    policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "elasticloadbalancing:*",
        "ec2:*",
        "ecs:*"
      ],
      "Resource": [
        "*"
      ]
    }
  ]
}
EOF
}

resource "aws_elb" "main" {
  availability_zones = ["us-west-2a"]

  listener {
    instance_port = %d
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }
}

resource "aws_ecs_service" "with_lb_changes" {
  name = "ghost"
  cluster = "${aws_ecs_cluster.main.id}"
  task_definition = "${aws_ecs_task_definition.with_lb_changes.arn}"
  desired_count = 1
  iam_role = "${aws_iam_role.ecs_service.name}"

  load_balancer {
    elb_name = "${aws_elb.main.id}"
    container_name = "%s"
    container_port = "%d"
  }

  depends_on = ["aws_iam_role_policy.ecs_service"]
}
`

var testAccAWSEcsService_withLbChanges = fmt.Sprintf(
	tpl_testAccAWSEcsService_withLbChanges,
	"ghost:latest", "ghost", 2368, 8080, 8080, "ghost", 2368)
var testAccAWSEcsService_withLbChanges_modified = fmt.Sprintf(
	tpl_testAccAWSEcsService_withLbChanges,
	"nginx:latest", "nginx", 80, 8080, 8080, "nginx", 80)

func testAccAWSEcsServiceWithFamilyAndRevision(rName string) string {
	return fmt.Sprintf(`
resource "aws_ecs_cluster" "default" {
	name = "%s"
}

resource "aws_ecs_task_definition" "jenkins" {
  family = "%s"
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
  name = "%s"
  cluster = "${aws_ecs_cluster.default.id}"
  task_definition = "${aws_ecs_task_definition.jenkins.family}:${aws_ecs_task_definition.jenkins.revision}"
  desired_count = 1
}`, rName, rName, rName)
}

func testAccAWSEcsServiceWithFamilyAndRevisionModified(rName string) string {
	return fmt.Sprintf(`
resource "aws_ecs_cluster" "default" {
	name = "%s"
}

resource "aws_ecs_task_definition" "jenkins" {
  family = "%s"
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
  name = "%s"
  cluster = "${aws_ecs_cluster.default.id}"
  task_definition = "${aws_ecs_task_definition.jenkins.family}:${aws_ecs_task_definition.jenkins.revision}"
  desired_count = 1
}`, rName, rName, rName)
}

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

var testAccAWSEcsServiceWithEcsClusterName = `
resource "aws_ecs_cluster" "default" {
	name = "terraformecstestcluster"
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
  cluster = "${aws_ecs_cluster.default.name}"
  task_definition = "${aws_ecs_task_definition.jenkins.arn}"
  desired_count = 1
}
`

func testAccAWSEcsServiceWithAlb(rName string) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {}

resource "aws_vpc" "main" {
  cidr_block = "10.10.0.0/16"
	tags {
		Name = "TestAccAWSEcsService_withAlb"
	}
}

resource "aws_subnet" "main" {
  count = 2
  cidr_block = "${cidrsubnet(aws_vpc.main.cidr_block, 8, count.index)}"
  availability_zone = "${data.aws_availability_zones.available.names[count.index]}"
  vpc_id = "${aws_vpc.main.id}"
}

resource "aws_ecs_cluster" "main" {
  name = "%s"
}

resource "aws_ecs_task_definition" "with_lb_changes" {
  family = "%s"
  container_definitions = <<DEFINITION
[
  {
    "cpu": 256,
    "essential": true,
    "image": "ghost:latest",
    "memory": 512,
    "name": "ghost",
    "portMappings": [
      {
        "containerPort": 2368,
        "hostPort": 8080
      }
    ]
  }
]
DEFINITION
}

resource "aws_iam_role" "ecs_service" {
    name = "%s"
    assume_role_policy = <<EOF
{
  "Version": "2008-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "ecs.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "ecs_service" {
    name = "%s"
    role = "${aws_iam_role.ecs_service.name}"
    policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:Describe*",
        "elasticloadbalancing:DeregisterInstancesFromLoadBalancer",
        "elasticloadbalancing:DeregisterTargets",
        "elasticloadbalancing:Describe*",
        "elasticloadbalancing:RegisterInstancesWithLoadBalancer",
        "elasticloadbalancing:RegisterTargets"
      ],
      "Resource": "*"
    }
  ]
}
EOF
}

resource "aws_alb_target_group" "test" {
  name = "%s"
  port = 80
  protocol = "HTTP"
  vpc_id = "${aws_vpc.main.id}"
}

resource "aws_alb" "main" {
  name            = "%s"
  internal        = true
  subnets         = ["${aws_subnet.main.*.id}"]
}

resource "aws_alb_listener" "front_end" {
  load_balancer_arn = "${aws_alb.main.id}"
  port = "80"
  protocol = "HTTP"

  default_action {
    target_group_arn = "${aws_alb_target_group.test.id}"
    type = "forward"
  }
}

resource "aws_ecs_service" "with_alb" {
  name = "%s"
  cluster = "${aws_ecs_cluster.main.id}"
  task_definition = "${aws_ecs_task_definition.with_lb_changes.arn}"
  desired_count = 1
  iam_role = "${aws_iam_role.ecs_service.name}"

  load_balancer {
    target_group_arn = "${aws_alb_target_group.test.id}"
    container_name = "ghost"
    container_port = "2368"
  }

  depends_on = [
    "aws_iam_role_policy.ecs_service",
    "aws_alb_listener.front_end"
  ]
}
`, rName, rName, rName, rName, rName, rName, rName)
}
