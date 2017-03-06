// make testacc TEST=./builtin/providers/aws/ TESTARGS='-run=TestAccDataSourceAwsNetworkAcl_'
package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceAwsNetworkAcl_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceAwsNetworkAclBasicConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.aws_network_acl.test_by_network_acl_id", "id",
						"aws_network_acl.bar", "id"),
					resource.TestCheckResourceAttrPair(
						"data.aws_network_acl.test_by_network_acl_id", "vpc_id",
						"aws_vpc.foo", "id"),
					resource.TestCheckResourceAttr(
						"data.aws_network_acl.test_by_network_acl_id", "tags.%", "1"),
					resource.TestCheckNoResourceAttr(
						"data.aws_network_acl.test_by_network_acl_id", "subnet_ids"),
					resource.TestCheckResourceAttr(
						"data.aws_network_acl.test_by_network_acl_id", "egress.#", "1"),
					resource.TestCheckResourceAttr(
						"data.aws_network_acl.test_by_network_acl_id", "egress.0.rule_no", "2"),
					resource.TestCheckResourceAttr(
						"data.aws_network_acl.test_by_network_acl_id", "egress.0.cidr_block", "10.3.0.0/18"),
					resource.TestCheckResourceAttr(
						"data.aws_network_acl.test_by_network_acl_id", "egress.0.ipv6_cidr_block", ""),
					resource.TestCheckResourceAttr(
						"data.aws_network_acl.test_by_network_acl_id", "ingress.#", "1"),
					resource.TestCheckResourceAttr(
						"data.aws_network_acl.test_by_network_acl_id", "ingress.0.protocol", "6"),
					resource.TestCheckResourceAttr(
						"data.aws_network_acl.test_by_network_acl_id", "ingress.0.from_port", "80"),
					resource.TestCheckResourceAttr(
						"data.aws_network_acl.test_by_network_acl_id", "ingress.0.to_port", "81"),
					resource.TestCheckResourceAttrPair(
						"data.aws_network_acl.test_by_name_tag", "network_acl_id",
						"aws_network_acl.bar", "id"),
					resource.TestCheckResourceAttr(
						"data.aws_network_acl.test_by_name_tag", "default", "false"),
				),
			},
		},
	})
}

func TestAccDataSourceAwsNetworkAcl_default(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceAwsNetworkAclDefaultConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.aws_network_acl.test_default", "id",
						"aws_vpc.foo", "default_network_acl_id"),
					resource.TestCheckResourceAttr(
						"data.aws_network_acl.test_default", "default", "true"),
				),
			},
		},
	})
}

func TestAccDataSourceAwsNetworkAcl_subnets(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceAwsNetworkAclSubnetsConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.aws_network_acl.test_one_subnet", "id",
						"aws_network_acl.bar", "id"),
					resource.TestCheckNoResourceAttr(
						"data.aws_network_acl.test_one_subnet", "ingress"),
					resource.TestCheckResourceAttr(
						"data.aws_network_acl.test_one_subnet", "subnet_ids.#", "2"),
					resource.TestCheckResourceAttrPair(
						"data.aws_network_acl.test_two_subnets", "network_acl_id",
						"aws_network_acl.bar", "id"),
					resource.TestCheckNoResourceAttr(
						"data.aws_network_acl.test_two_subnets", "egress"),
				),
			},
		},
	})
}

const testAccDataSourceAwsNetworkAclBasicConfig = `
provider "aws" {
  region = "us-west-2"
}

resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"

  tags {
    Name = "terraform-testacc-nacl-data-source-foo"
  }
}

resource "aws_network_acl" "bar" {
	vpc_id = "${aws_vpc.foo.id}"

	egress {
			protocol = "tcp"
			rule_no = 2
			action = "allow"
			cidr_block =  "10.3.0.0/18"
			from_port = 443
			to_port = 443
	}

	ingress {
			protocol = "tcp"
			rule_no = 1
			action = "allow"
			cidr_block =  "10.3.0.0/18"
			from_port = 80
			to_port = 81
	}

	tags {
			Name = "terraform-testacc-nacl-data-source-bar"
	}
}

data "aws_network_acl" "test_by_network_acl_id" {
	network_acl_id = "${aws_network_acl.bar.id}"
}

data "aws_network_acl" "test_by_name_tag" {
	vpc_id = "${aws_network_acl.bar.vpc_id}"

	tags {
			Name = "terraform-testacc-nacl-data-source-bar"
	}
}
`

const testAccDataSourceAwsNetworkAclDefaultConfig = `
provider "aws" {
  region = "us-west-2"
}

resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"

  tags {
    Name = "terraform-testacc-nacl-data-source-foo"
  }
}

data "aws_network_acl" "test_default" {
	vpc_id  = "${aws_vpc.foo.id}"
	default = true
}
`

const testAccDataSourceAwsNetworkAclSubnetsConfig = `
provider "aws" {
  region = "us-west-2"
}

resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"

  tags {
    Name = "terraform-testacc-nacl-data-source-foo"
  }
}

resource "aws_subnet" "baz" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"

  tags {
    Name = "terraform-testacc-nacl-data-source-baz"
  }
}

resource "aws_subnet" "quux" {
	cidr_block = "10.1.111.0/24"
	vpc_id = "${aws_vpc.foo.id}"

  tags {
    Name = "terraform-testacc-nacl-data-source-quux"
  }
}

resource "aws_network_acl" "bar" {
	vpc_id = "${aws_vpc.foo.id}"
	subnet_ids = [
		"${aws_subnet.baz.id}",
		"${aws_subnet.quux.id}",
	]

	tags {
			Name = "terraform-testacc-nacl-data-source-bar"
	}
}

data "aws_network_acl" "test_one_subnet" {
	vpc_id = "${aws_network_acl.bar.vpc_id}"

	subnet_ids = [
		"${aws_subnet.baz.id}",
	]
}

data "aws_network_acl" "test_two_subnets" {
	vpc_id = "${aws_network_acl.bar.vpc_id}"

	subnet_ids = [
		"${aws_subnet.baz.id}",
		"${aws_subnet.quux.id}",
	]
}
`
