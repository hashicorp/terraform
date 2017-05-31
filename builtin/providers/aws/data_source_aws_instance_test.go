package aws

import (
	"testing"

	"fmt"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSInstanceDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.aws_instance.web-instance", "ami", "ami-4fccb37f"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.web-instance", "tags.#", "1"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.web-instance", "instance_type", "m1.small"),
				),
			},
		},
	})
}

func TestAccAWSInstanceDataSource_tags(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceDataSourceConfig_Tags(rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.aws_instance.web-instance", "ami", "ami-4fccb37f"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.web-instance", "tags.#", "2"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.web-instance", "instance_type", "m1.small"),
				),
			},
		},
	})
}

func TestAccAWSInstanceDataSource_AzUserData(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceDataSourceConfig_AzUserData,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "ami", "ami-4fccb37f"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "tags.#", "1"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "instance_type", "m1.small"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "availability_zone", "us-west-2a"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "user_data", "3dc39dda39be1205215e776bad998da361a5955d"),
				),
			},
		},
	})
}

func TestAccAWSInstanceDataSource_gp2IopsDevice(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceDataSourceConfig_gp2IopsDevice,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "ami", "ami-55a7ea65"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "instance_type", "m3.medium"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.0.volume_size", "11"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.0.volume_type", "gp2"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.0.iops", "100"),
				),
			},
		},
	})
}

func TestAccAWSInstanceDataSource_blockDevices(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceDataSourceConfig_blockDevices,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "ami", "ami-55a7ea65"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "instance_type", "m3.medium"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.0.volume_size", "11"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.0.volume_type", "gp2"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_block_device.#", "3"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ephemeral_block_device.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSInstanceDataSource_rootInstanceStore(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceDataSourceConfig_rootInstanceStore,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "ami", "ami-44c36524"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "instance_type", "m3.medium"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_block_device.#", "0"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_optimized", "false"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.#", "0"),
				),
			},
		},
	})
}

func TestAccAWSInstanceDataSource_privateIP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceDataSourceConfig_privateIP,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "ami", "ami-c5eabbf5"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "instance_type", "t2.micro"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "private_ip", "10.1.1.42"),
				),
			},
		},
	})
}

func TestAccAWSInstanceDataSource_keyPair(t *testing.T) {
	rName := fmt.Sprintf("tf-test-key-%d", acctest.RandInt())
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceDataSourceConfig_keyPair(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "ami", "ami-408c7f28"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "instance_type", "t1.micro"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "tags.#", "1"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "key_name", rName),
				),
			},
		},
	})
}

func TestAccAWSInstanceDataSource_VPC(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceDataSourceConfig_VPC,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "ami", "ami-4fccb37f"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "instance_type", "m1.small"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "user_data", "562a3e32810edf6ff09994f050f12e799452379d"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "associate_public_ip_address", "true"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "tenancy", "dedicated"),
				),
			},
		},
	})
}

func TestAccAWSInstanceDataSource_SecurityGroups(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceDataSourceConfig_SecurityGroups(rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "ami", "ami-408c7f28"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "instance_type", "m1.small"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "vpc_security_group_ids.#", "0"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "security_groups.#", "1"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "user_data", "3dc39dda39be1205215e776bad998da361a5955d"),
				),
			},
		},
	})
}

func TestAccAWSInstanceDataSource_VPCSecurityGroups(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceDataSourceConfig_VPCSecurityGroups,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "ami", "ami-21f78e11"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "instance_type", "t1.micro"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "security_groups.#", "0"),
					resource.TestCheckResourceAttr(
						"data.aws_instance.foo", "vpc_security_group_ids.#", "1"),
				),
			},
		},
	})
}

// Lookup based on InstanceID
const testAccInstanceDataSourceConfig = `
resource "aws_instance" "web" {
	# us-west-2
  ami = "ami-4fccb37f"
  instance_type = "m1.small"
  tags {
    Name = "HelloWorld"
  }
}

data "aws_instance" "web-instance" {
  filter {
    name = "instance-id"
    values = ["${aws_instance.web.id}"]
  }
}
`

// Use the tags attribute to filter
func testAccInstanceDataSourceConfig_Tags(rInt int) string {
	return fmt.Sprintf(`
resource "aws_instance" "web" {
	# us-west-2
  ami = "ami-4fccb37f"
  instance_type = "m1.small"
  tags {
    Name = "HelloWorld"
    TestSeed = "%d"
  }
}

data "aws_instance" "web-instance" {
  instance_tags {
    Name = "${aws_instance.web.tags["Name"]}"
    TestSeed = "%d"
  }
}
`, rInt, rInt)
}

// filter on tag, populate more attributes
const testAccInstanceDataSourceConfig_AzUserData = `
resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-4fccb37f"
	availability_zone = "us-west-2a"

	instance_type = "m1.small"
	user_data = "foo:-with-character's"
	tags {
	  TFAccTest = "YesThisIsATest"
	}
}

data "aws_instance" "foo" {
  instance_id = "${aws_instance.foo.id}"
}
`

// GP2IopsDevice
const testAccInstanceDataSourceConfig_gp2IopsDevice = `
resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-55a7ea65"
	instance_type = "m3.medium"
	root_block_device {
		volume_type = "gp2"
		volume_size = 11
	}
}

data "aws_instance" "foo" {
  instance_id = "${aws_instance.foo.id}"
}
`

// Block Device
const testAccInstanceDataSourceConfig_blockDevices = `
resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-55a7ea65"
	instance_type = "m3.medium"

	root_block_device {
		volume_type = "gp2"
		volume_size = 11
	}
	ebs_block_device {
		device_name = "/dev/sdb"
		volume_size = 9
	}
	ebs_block_device {
		device_name = "/dev/sdc"
		volume_size = 10
		volume_type = "io1"
		iops = 100
	}

	# Encrypted ebs block device
	ebs_block_device {
		device_name = "/dev/sdd"
		volume_size = 12
		encrypted = true
	}

	ephemeral_block_device {
		device_name = "/dev/sde"
		virtual_name = "ephemeral0"
	}
}

data "aws_instance" "foo" {
  instance_id = "${aws_instance.foo.id}"
}
`

const testAccInstanceDataSourceConfig_rootInstanceStore = `
resource "aws_instance" "foo" {
  ami = "ami-44c36524"
  instance_type = "m3.medium"
}
data "aws_instance" "foo" {
  instance_id = "${aws_instance.foo.id}"
}
`

const testAccInstanceDataSourceConfig_privateIP = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "testAccInstanceDataSourceConfig_privateIP"
	}
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_instance" "foo" {
	ami = "ami-c5eabbf5"
	instance_type = "t2.micro"
	subnet_id = "${aws_subnet.foo.id}"
	private_ip = "10.1.1.42"
}

data "aws_instance" "foo" {
  instance_id = "${aws_instance.foo.id}"
}
`

func testAccInstanceDataSourceConfig_keyPair(rName string) string {
	return fmt.Sprintf(`
provider "aws" {
	region = "us-east-1"
}

resource "aws_key_pair" "debugging" {
	key_name = "%s"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}

resource "aws_instance" "foo" {
  ami = "ami-408c7f28"
  instance_type = "t1.micro"
  key_name = "${aws_key_pair.debugging.key_name}"
	tags {
		Name = "testAccInstanceDataSourceConfigKeyPair_TestAMI"
	}
}

data "aws_instance" "foo" {
  filter {
    name = "tag:Name"
    values = ["testAccInstanceDataSourceConfigKeyPair_TestAMI"]
  }
  filter {
    name = "key-name"
    values = ["${aws_instance.foo.key_name}"]
  }
}`, rName)
}

const testAccInstanceDataSourceConfig_VPC = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "testAccInstanceDataSourceConfig_VPC"
	}
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
	subnet_id = "${aws_subnet.foo.id}"
	associate_public_ip_address = true
	tenancy = "dedicated"
	# pre-encoded base64 data
	user_data = "3dc39dda39be1205215e776bad998da361a5955d"
}

data "aws_instance" "foo" {
  instance_id = "${aws_instance.foo.id}"
}
`

func testAccInstanceDataSourceConfig_SecurityGroups(rInt int) string {
	return fmt.Sprintf(`
provider "aws" {
	region = "us-east-1"
}

resource "aws_security_group" "tf_test_foo" {
	name = "tf_test_foo-%d"
	description = "foo"

	ingress {
		protocol = "icmp"
		from_port = -1
		to_port = -1
		cidr_blocks = ["0.0.0.0/0"]
	}
}

resource "aws_instance" "foo" {
	ami = "ami-408c7f28"
	instance_type = "m1.small"
	security_groups = ["${aws_security_group.tf_test_foo.name}"]
	user_data = "foo:-with-character's"
}

data "aws_instance" "foo" {
  instance_id = "${aws_instance.foo.id}"
}
`, rInt)
}

const testAccInstanceDataSourceConfig_VPCSecurityGroups = `
resource "aws_internet_gateway" "gw" {
  vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"
	tags {
		Name = "tf-network-test"
	}
}

resource "aws_security_group" "tf_test_foo" {
  name = "tf_test_foo"
  description = "foo"
  vpc_id="${aws_vpc.foo.id}"

  ingress {
    protocol = "icmp"
    from_port = -1
    to_port = -1
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_subnet" "foo" {
  cidr_block = "10.1.1.0/24"
  vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_instance" "foo_instance" {
  ami = "ami-21f78e11"
  instance_type = "t1.micro"
  vpc_security_group_ids = ["${aws_security_group.tf_test_foo.id}"]
  subnet_id = "${aws_subnet.foo.id}"
	depends_on = ["aws_internet_gateway.gw"]
}

data "aws_instance" "foo" {
  instance_id = "${aws_instance.foo_instance.id}"
}
`
