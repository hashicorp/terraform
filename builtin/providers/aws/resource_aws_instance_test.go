package aws

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSInstance_basic(t *testing.T) {
	var v ec2.Instance
	var vol *ec2.Volume

	testCheck := func(*terraform.State) error {
		if *v.Placement.AvailabilityZone != "us-west-2a" {
			return fmt.Errorf("bad availability zone: %#v", *v.Placement.AvailabilityZone)
		}

		if len(v.SecurityGroups) == 0 {
			return fmt.Errorf("no security groups: %#v", v.SecurityGroups)
		}
		if *v.SecurityGroups[0].GroupName != "tf_test_foo" {
			return fmt.Errorf("no security groups: %#v", v.SecurityGroups)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			// Create a volume to cover #1249
			resource.TestStep{
				// Need a resource in this config so the provisioner will be available
				Config: testAccInstanceConfig_pre,
				Check: func(*terraform.State) error {
					conn := testAccProvider.Meta().(*AWSClient).ec2conn
					var err error
					vol, err = conn.CreateVolume(&ec2.CreateVolumeInput{
						AvailabilityZone: aws.String("us-west-2a"),
						Size:             aws.Int64(int64(5)),
					})
					return err
				},
			},

			resource.TestStep{
				Config: testAccInstanceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"aws_instance.foo", &v),
					testCheck,
					resource.TestCheckResourceAttr(
						"aws_instance.foo",
						"user_data",
						"3dc39dda39be1205215e776bad998da361a5955d"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_block_device.#", "0"),
				),
			},

			// We repeat the exact same test so that we can be sure
			// that the user data hash stuff is working without generating
			// an incorrect diff.
			resource.TestStep{
				Config: testAccInstanceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"aws_instance.foo", &v),
					testCheck,
					resource.TestCheckResourceAttr(
						"aws_instance.foo",
						"user_data",
						"3dc39dda39be1205215e776bad998da361a5955d"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_block_device.#", "0"),
				),
			},

			// Clean up volume created above
			resource.TestStep{
				Config: testAccInstanceConfig,
				Check: func(*terraform.State) error {
					conn := testAccProvider.Meta().(*AWSClient).ec2conn
					_, err := conn.DeleteVolume(&ec2.DeleteVolumeInput{VolumeId: vol.VolumeId})
					return err
				},
			},
		},
	})
}

func TestAccAWSInstance_blockDevices(t *testing.T) {
	var v ec2.Instance

	testCheck := func() resource.TestCheckFunc {
		return func(*terraform.State) error {

			// Map out the block devices by name, which should be unique.
			blockDevices := make(map[string]*ec2.InstanceBlockDeviceMapping)
			for _, blockDevice := range v.BlockDeviceMappings {
				blockDevices[*blockDevice.DeviceName] = blockDevice
			}

			// Check if the root block device exists.
			if _, ok := blockDevices["/dev/sda1"]; !ok {
				fmt.Errorf("block device doesn't exist: /dev/sda1")
			}

			// Check if the secondary block device exists.
			if _, ok := blockDevices["/dev/sdb"]; !ok {
				fmt.Errorf("block device doesn't exist: /dev/sdb")
			}

			// Check if the third block device exists.
			if _, ok := blockDevices["/dev/sdc"]; !ok {
				fmt.Errorf("block device doesn't exist: /dev/sdc")
			}

			// Check if the encrypted block device exists
			if _, ok := blockDevices["/dev/sdd"]; !ok {
				fmt.Errorf("block device doesn't exist: /dev/sdd")
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceConfigBlockDevices,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"aws_instance.foo", &v),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.0.volume_size", "11"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.0.volume_type", "gp2"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_block_device.#", "3"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_block_device.2576023345.device_name", "/dev/sdb"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_block_device.2576023345.volume_size", "9"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_block_device.2576023345.volume_type", "standard"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_block_device.2554893574.device_name", "/dev/sdc"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_block_device.2554893574.volume_size", "10"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_block_device.2554893574.volume_type", "io1"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_block_device.2554893574.iops", "100"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_block_device.2634515331.device_name", "/dev/sdd"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_block_device.2634515331.encrypted", "true"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_block_device.2634515331.volume_size", "12"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ephemeral_block_device.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ephemeral_block_device.1692014856.device_name", "/dev/sde"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ephemeral_block_device.1692014856.virtual_name", "ephemeral0"),
					testCheck(),
				),
			},
		},
	})
}

func TestAccAWSInstance_sourceDestCheck(t *testing.T) {
	var v ec2.Instance

	testCheck := func(enabled bool) resource.TestCheckFunc {
		return func(*terraform.State) error {
			if v.SourceDestCheck == nil {
				return fmt.Errorf("bad source_dest_check: got nil")
			}
			if *v.SourceDestCheck != enabled {
				return fmt.Errorf("bad source_dest_check: %#v", *v.SourceDestCheck)
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceConfigSourceDestDisable,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testCheck(false),
				),
			},

			resource.TestStep{
				Config: testAccInstanceConfigSourceDestEnable,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testCheck(true),
				),
			},

			resource.TestStep{
				Config: testAccInstanceConfigSourceDestDisable,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testCheck(false),
				),
			},
		},
	})
}

func TestAccAWSInstance_disableApiTermination(t *testing.T) {
	var v ec2.Instance

	checkDisableApiTermination := func(expected bool) resource.TestCheckFunc {
		return func(*terraform.State) error {
			conn := testAccProvider.Meta().(*AWSClient).ec2conn
			r, err := conn.DescribeInstanceAttribute(&ec2.DescribeInstanceAttributeInput{
				InstanceId: v.InstanceId,
				Attribute:  aws.String("disableApiTermination"),
			})
			if err != nil {
				return err
			}
			got := *r.DisableApiTermination.Value
			if got != expected {
				return fmt.Errorf("expected: %t, got: %t", expected, got)
			}
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceConfigDisableAPITermination(true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					checkDisableApiTermination(true),
				),
			},

			resource.TestStep{
				Config: testAccInstanceConfigDisableAPITermination(false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					checkDisableApiTermination(false),
				),
			},
		},
	})
}

func TestAccAWSInstance_vpc(t *testing.T) {
	var v ec2.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceConfigVPC,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"aws_instance.foo", &v),
				),
			},
		},
	})
}

func TestAccAWSInstance_multipleRegions(t *testing.T) {
	var v ec2.Instance

	// record the initialized providers so that we can use them to
	// check for the instances in each region
	var providers []*schema.Provider
	providerFactories := map[string]terraform.ResourceProviderFactory{
		"aws": func() (terraform.ResourceProvider, error) {
			p := Provider()
			providers = append(providers, p.(*schema.Provider))
			return p, nil
		},
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckInstanceDestroyWithProviders(&providers),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceConfigMultipleRegions,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExistsWithProviders(
						"aws_instance.foo", &v, &providers),
					testAccCheckInstanceExistsWithProviders(
						"aws_instance.bar", &v, &providers),
				),
			},
		},
	})
}

func TestAccAWSInstance_NetworkInstanceSecurityGroups(t *testing.T) {
	var v ec2.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceNetworkInstanceSecurityGroups,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"aws_instance.foo_instance", &v),
				),
			},
		},
	})
}

func TestAccAWSInstance_NetworkInstanceVPCSecurityGroupIDs(t *testing.T) {
	var v ec2.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceNetworkInstanceVPCSecurityGroupIDs,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"aws_instance.foo_instance", &v),
					resource.TestCheckResourceAttr(
						"aws_instance.foo_instance", "security_groups.#", "0"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo_instance", "vpc_security_group_ids.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSInstance_tags(t *testing.T) {
	var v ec2.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckInstanceConfigTags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testAccCheckTags(&v.Tags, "foo", "bar"),
					// Guard against regression of https://github.com/hashicorp/terraform/issues/914
					testAccCheckTags(&v.Tags, "#", ""),
				),
			},

			resource.TestStep{
				Config: testAccCheckInstanceConfigTagsUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testAccCheckTags(&v.Tags, "foo", ""),
					testAccCheckTags(&v.Tags, "bar", "baz"),
				),
			},
		},
	})
}

func TestAccAWSInstance_privateIP(t *testing.T) {
	var v ec2.Instance

	testCheckPrivateIP := func() resource.TestCheckFunc {
		return func(*terraform.State) error {
			if *v.PrivateIpAddress != "10.1.1.42" {
				return fmt.Errorf("bad private IP: %s", *v.PrivateIpAddress)
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceConfigPrivateIP,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testCheckPrivateIP(),
				),
			},
		},
	})
}

func TestAccAWSInstance_associatePublicIPAndPrivateIP(t *testing.T) {
	var v ec2.Instance

	testCheckPrivateIP := func() resource.TestCheckFunc {
		return func(*terraform.State) error {
			if *v.PrivateIpAddress != "10.1.1.42" {
				return fmt.Errorf("bad private IP: %s", *v.PrivateIpAddress)
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceConfigAssociatePublicIPAndPrivateIP,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testCheckPrivateIP(),
				),
			},
		},
	})
}

// Guard against regression with KeyPairs
// https://github.com/hashicorp/terraform/issues/2302
func TestAccAWSInstance_keyPairCheck(t *testing.T) {
	var v ec2.Instance

	testCheckKeyPair := func(keyName string) resource.TestCheckFunc {
		return func(*terraform.State) error {
			if v.KeyName == nil {
				return fmt.Errorf("No Key Pair found, expected(%s)", keyName)
			}
			if v.KeyName != nil && *v.KeyName != keyName {
				return fmt.Errorf("Bad key name, expected (%s), got (%s)", keyName, *v.KeyName)
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceConfigKeyPair,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testCheckKeyPair("tmp-key"),
				),
			},
		},
	})
}

func TestAccAWSInstance_rootBlockDeviceMismatch(t *testing.T) {
	var v ec2.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceConfigRootBlockDeviceMismatch,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.0.volume_size", "13"),
				),
			},
		},
	})
}

func testAccCheckInstanceDestroy(s *terraform.State) error {
	return testAccCheckInstanceDestroyWithProvider(s, testAccProvider)
}

func testAccCheckInstanceDestroyWithProviders(providers *[]*schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, provider := range *providers {
			if provider.Meta() == nil {
				continue
			}
			if err := testAccCheckInstanceDestroyWithProvider(s, provider); err != nil {
				return err
			}
		}
		return nil
	}
}

func testAccCheckInstanceDestroyWithProvider(s *terraform.State, provider *schema.Provider) error {
	conn := provider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_instance" {
			continue
		}

		// Try to find the resource
		var err error
		resp, err := conn.DescribeInstances(&ec2.DescribeInstancesInput{
			InstanceIds: []*string{aws.String(rs.Primary.ID)},
		})
		if err == nil {
			if len(resp.Reservations) > 0 {
				return fmt.Errorf("still exist.")
			}

			return nil
		}

		// Verify the error is what we want
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "InvalidInstanceID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckInstanceExists(n string, i *ec2.Instance) resource.TestCheckFunc {
	providers := []*schema.Provider{testAccProvider}
	return testAccCheckInstanceExistsWithProviders(n, i, &providers)
}

func testAccCheckInstanceExistsWithProviders(n string, i *ec2.Instance, providers *[]*schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		for _, provider := range *providers {
			// Ignore if Meta is empty, this can happen for validation providers
			if provider.Meta() == nil {
				continue
			}

			conn := provider.Meta().(*AWSClient).ec2conn
			resp, err := conn.DescribeInstances(&ec2.DescribeInstancesInput{
				InstanceIds: []*string{aws.String(rs.Primary.ID)},
			})
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidInstanceID.NotFound" {
				continue
			}
			if err != nil {
				return err
			}

			if len(resp.Reservations) > 0 {
				*i = *resp.Reservations[0].Instances[0]
				return nil
			}
		}

		return fmt.Errorf("Instance not found")
	}
}

func TestInstanceTenancySchema(t *testing.T) {
	actualSchema := resourceAwsInstance().Schema["tenancy"]
	expectedSchema := &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
		Computed: true,
		ForceNew: true,
	}
	if !reflect.DeepEqual(actualSchema, expectedSchema) {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			actualSchema,
			expectedSchema)
	}
}

const testAccInstanceConfig_pre = `
resource "aws_security_group" "tf_test_foo" {
	name = "tf_test_foo"
	description = "foo"

	ingress {
		protocol = "icmp"
		from_port = -1
		to_port = -1
		cidr_blocks = ["0.0.0.0/0"]
	}
}
`

const testAccInstanceConfig = `
resource "aws_security_group" "tf_test_foo" {
	name = "tf_test_foo"
	description = "foo"

	ingress {
		protocol = "icmp"
		from_port = -1
		to_port = -1
		cidr_blocks = ["0.0.0.0/0"]
	}
}

resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-4fccb37f"
	availability_zone = "us-west-2a"

	instance_type = "m1.small"
	security_groups = ["${aws_security_group.tf_test_foo.name}"]
	user_data = "foo:-with-character's"
}
`

const testAccInstanceConfigBlockDevices = `
resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-55a7ea65"

	# In order to attach an encrypted volume to an instance you need to have an
	# m3.medium or larger. See "Supported Instance Types" in:
	# http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/EBSEncryption.html
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
`

const testAccInstanceConfigSourceDestEnable = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
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
}
`

const testAccInstanceConfigSourceDestDisable = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
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
	source_dest_check = false
}
`

func testAccInstanceConfigDisableAPITermination(val bool) string {
	return fmt.Sprintf(`
	resource "aws_vpc" "foo" {
		cidr_block = "10.1.0.0/16"
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
		disable_api_termination = %t
	}
	`, val)
}

const testAccInstanceConfigVPC = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
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
}
`

const testAccInstanceConfigMultipleRegions = `
provider "aws" {
	alias = "west"
	region = "us-west-2"
}

provider "aws" {
	alias = "east"
	region = "us-east-1"
}

resource "aws_instance" "foo" {
	# us-west-2
	provider = "aws.west"
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
}

resource "aws_instance" "bar" {
	# us-east-1
	provider = "aws.east"
	ami = "ami-8c6ea9e4"
	instance_type = "m1.small"
}
`

const testAccCheckInstanceConfigTags = `
resource "aws_instance" "foo" {
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
	tags {
		foo = "bar"
	}
}
`

const testAccCheckInstanceConfigTagsUpdate = `
resource "aws_instance" "foo" {
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
	tags {
		bar = "baz"
	}
}
`

const testAccInstanceConfigPrivateIP = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
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
`

const testAccInstanceConfigAssociatePublicIPAndPrivateIP = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_instance" "foo" {
	ami = "ami-c5eabbf5"
	instance_type = "t2.micro"
	subnet_id = "${aws_subnet.foo.id}"
	associate_public_ip_address = true
	private_ip = "10.1.1.42"
}
`

const testAccInstanceNetworkInstanceSecurityGroups = `
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
  security_groups = ["${aws_security_group.tf_test_foo.id}"]
  subnet_id = "${aws_subnet.foo.id}"
  associate_public_ip_address = true
	depends_on = ["aws_internet_gateway.gw"]
}

resource "aws_eip" "foo_eip" {
  instance = "${aws_instance.foo_instance.id}"
  vpc = true
	depends_on = ["aws_internet_gateway.gw"]
}
`

const testAccInstanceNetworkInstanceVPCSecurityGroupIDs = `
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

resource "aws_eip" "foo_eip" {
  instance = "${aws_instance.foo_instance.id}"
  vpc = true
	depends_on = ["aws_internet_gateway.gw"]
}
`

const testAccInstanceConfigKeyPair = `
provider "aws" {
	region = "us-east-1"
}

resource "aws_key_pair" "debugging" {
	key_name = "tmp-key"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}

resource "aws_instance" "foo" {
  ami = "ami-408c7f28"
  instance_type = "t1.micro"
  key_name = "${aws_key_pair.debugging.key_name}"
}
`

const testAccInstanceConfigRootBlockDeviceMismatch = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_instance" "foo" {
	// This is an AMI with RootDeviceName: "/dev/sda1"; actual root: "/dev/sda"
	ami = "ami-ef5b69df"
	instance_type = "t1.micro"
	subnet_id = "${aws_subnet.foo.id}"
	root_block_device {
		volume_size = 13
	}
}
`
