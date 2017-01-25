package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceAWSALB_basic(t *testing.T) {
	albName := fmt.Sprintf("testaccawsalb-basic-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAWSALBConfigBasic(albName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.aws_alb.alb_test_with_arn", "name", albName),
					resource.TestCheckResourceAttr("data.aws_alb.alb_test_with_arn", "internal", "true"),
					resource.TestCheckResourceAttr("data.aws_alb.alb_test_with_arn", "subnets.#", "2"),
					resource.TestCheckResourceAttr("data.aws_alb.alb_test_with_arn", "security_groups.#", "1"),
					resource.TestCheckResourceAttr("data.aws_alb.alb_test_with_arn", "tags.%", "1"),
					resource.TestCheckResourceAttr("data.aws_alb.alb_test_with_arn", "tags.TestName", "TestAccAWSALB_basic"),
					resource.TestCheckResourceAttr("data.aws_alb.alb_test_with_arn", "enable_deletion_protection", "false"),
					resource.TestCheckResourceAttr("data.aws_alb.alb_test_with_arn", "idle_timeout", "30"),
					resource.TestCheckResourceAttrSet("data.aws_alb.alb_test_with_arn", "vpc_id"),
					resource.TestCheckResourceAttrSet("data.aws_alb.alb_test_with_arn", "zone_id"),
					resource.TestCheckResourceAttrSet("data.aws_alb.alb_test_with_arn", "dns_name"),
					resource.TestCheckResourceAttrSet("data.aws_alb.alb_test_with_arn", "arn"),
					resource.TestCheckResourceAttr("data.aws_alb.alb_test_with_name", "name", albName),
					resource.TestCheckResourceAttr("data.aws_alb.alb_test_with_name", "internal", "true"),
					resource.TestCheckResourceAttr("data.aws_alb.alb_test_with_name", "subnets.#", "2"),
					resource.TestCheckResourceAttr("data.aws_alb.alb_test_with_name", "security_groups.#", "1"),
					resource.TestCheckResourceAttr("data.aws_alb.alb_test_with_name", "tags.%", "1"),
					resource.TestCheckResourceAttr("data.aws_alb.alb_test_with_name", "tags.TestName", "TestAccAWSALB_basic"),
					resource.TestCheckResourceAttr("data.aws_alb.alb_test_with_name", "enable_deletion_protection", "false"),
					resource.TestCheckResourceAttr("data.aws_alb.alb_test_with_name", "idle_timeout", "30"),
					resource.TestCheckResourceAttrSet("data.aws_alb.alb_test_with_name", "vpc_id"),
					resource.TestCheckResourceAttrSet("data.aws_alb.alb_test_with_name", "zone_id"),
					resource.TestCheckResourceAttrSet("data.aws_alb.alb_test_with_name", "dns_name"),
					resource.TestCheckResourceAttrSet("data.aws_alb.alb_test_with_name", "arn"),
				),
			},
		},
	})
}

func testAccDataSourceAWSALBConfigBasic(albName string) string {
	return fmt.Sprintf(`resource "aws_alb" "alb_test" {
  name            = "%s"
  internal        = true
  security_groups = ["${aws_security_group.alb_test.id}"]
  subnets         = ["${aws_subnet.alb_test.*.id}"]

  idle_timeout = 30
  enable_deletion_protection = false

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

variable "subnets" {
  default = ["10.0.1.0/24", "10.0.2.0/24"]
  type    = "list"
}

data "aws_availability_zones" "available" {}

resource "aws_vpc" "alb_test" {
  cidr_block = "10.0.0.0/16"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_subnet" "alb_test" {
  count                   = 2
  vpc_id                  = "${aws_vpc.alb_test.id}"
  cidr_block              = "${element(var.subnets, count.index)}"
  map_public_ip_on_launch = true
  availability_zone       = "${element(data.aws_availability_zones.available.names, count.index)}"

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

resource "aws_security_group" "alb_test" {
  name        = "allow_all_alb_test"
  description = "Used for ALB Testing"
  vpc_id      = "${aws_vpc.alb_test.id}"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    TestName = "TestAccAWSALB_basic"
  }
}

data "aws_alb" "alb_test_with_arn" {
	arn = "${aws_alb.alb_test.arn}"
}

data "aws_alb" "alb_test_with_name" {
	name = "${aws_alb.alb_test.name}"
}`, albName)
}
