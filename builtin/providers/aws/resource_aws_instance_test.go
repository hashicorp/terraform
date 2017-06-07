package aws

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/acctest"
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
		PreCheck: func() { testAccPreCheck(t) },

		// We ignore security groups because even with EC2 classic
		// we'll import as VPC security groups, which is fine. We verify
		// VPC security group import in other tests
		IDRefreshName:   "aws_instance.foo",
		IDRefreshIgnore: []string{"security_groups", "vpc_security_group_ids"},

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			// Create a volume to cover #1249
			{
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

			{
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
			{
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
			{
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

func TestAccAWSInstance_GP2IopsDevice(t *testing.T) {
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
				return fmt.Errorf("block device doesn't exist: /dev/sda1")
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_instance.foo",
		IDRefreshIgnore: []string{
			"ephemeral_block_device", "user_data", "security_groups", "vpc_security_groups"},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceGP2IopsDevice,
				//Config: testAccInstanceConfigBlockDevices,
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
						"aws_instance.foo", "root_block_device.0.iops", "100"),
					testCheck(),
				),
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
				return fmt.Errorf("block device doesn't exist: /dev/sda1")
			}

			// Check if the secondary block device exists.
			if _, ok := blockDevices["/dev/sdb"]; !ok {
				return fmt.Errorf("block device doesn't exist: /dev/sdb")
			}

			// Check if the third block device exists.
			if _, ok := blockDevices["/dev/sdc"]; !ok {
				return fmt.Errorf("block device doesn't exist: /dev/sdc")
			}

			// Check if the encrypted block device exists
			if _, ok := blockDevices["/dev/sdd"]; !ok {
				return fmt.Errorf("block device doesn't exist: /dev/sdd")
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_instance.foo",
		IDRefreshIgnore: []string{
			"ephemeral_block_device", "security_groups", "vpc_security_groups"},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
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

func TestAccAWSInstance_rootInstanceStore(t *testing.T) {
	var v ec2.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_instance.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "aws_instance" "foo" {
						# us-west-2
						# Amazon Linux HVM Instance Store 64-bit (2016.09.0)
						# https://aws.amazon.com/amazon-linux-ami
						ami = "ami-44c36524"

						# Only certain instance types support ephemeral root instance stores.
						# http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/InstanceStorage.html
						instance_type = "m3.medium"
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"aws_instance.foo", &v),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ami", "ami-44c36524"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_block_device.#", "0"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_optimized", "false"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "instance_type", "m3.medium"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.#", "0"),
				),
			},
		},
	})
}

func TestAcctABSInstance_noAMIEphemeralDevices(t *testing.T) {
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
				return fmt.Errorf("block device doesn't exist: /dev/sda1")
			}

			// Check if the secondary block not exists.
			if _, ok := blockDevices["/dev/sdb"]; ok {
				return fmt.Errorf("block device exist: /dev/sdb")
			}

			// Check if the third block device not exists.
			if _, ok := blockDevices["/dev/sdc"]; ok {
				return fmt.Errorf("block device exist: /dev/sdc")
			}
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_instance.foo",
		IDRefreshIgnore: []string{
			"ephemeral_block_device", "security_groups", "vpc_security_groups"},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "aws_instance" "foo" {
						# us-west-2
						ami = "ami-01f05461"  // This AMI (Ubuntu) contains two ephemerals

						instance_type = "c3.large"

						root_block_device {
							volume_type = "gp2"
							volume_size = 11
						}
						ephemeral_block_device {
							device_name = "/dev/sdb"
							no_device = true
						}
						ephemeral_block_device {
							device_name = "/dev/sdc"
							no_device = true
						}
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"aws_instance.foo", &v),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ami", "ami-01f05461"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_optimized", "false"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "instance_type", "c3.large"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.0.volume_size", "11"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.0.volume_type", "gp2"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ebs_block_device.#", "0"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ephemeral_block_device.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ephemeral_block_device.172787947.device_name", "/dev/sdb"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ephemeral_block_device.172787947.no_device", "true"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ephemeral_block_device.3336996981.device_name", "/dev/sdc"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "ephemeral_block_device.3336996981.no_device", "true"),
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
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_instance.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceConfigSourceDestDisable,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testCheck(false),
				),
			},

			{
				Config: testAccInstanceConfigSourceDestEnable,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testCheck(true),
				),
			},

			{
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
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_instance.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceConfigDisableAPITermination(true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					checkDisableApiTermination(true),
				),
			},

			{
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
		PreCheck:        func() { testAccPreCheck(t) },
		IDRefreshName:   "aws_instance.foo",
		IDRefreshIgnore: []string{"associate_public_ip_address"},
		Providers:       testAccProviders,
		CheckDestroy:    testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceConfigVPC,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"aws_instance.foo", &v),
					resource.TestCheckResourceAttr(
						"aws_instance.foo",
						"user_data",
						"562a3e32810edf6ff09994f050f12e799452379d"),
				),
			},
		},
	})
}

func TestAccAWSInstance_ipv6_supportAddressCount(t *testing.T) {
	var v ec2.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceConfigIpv6Support,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"aws_instance.foo", &v),
					resource.TestCheckResourceAttr(
						"aws_instance.foo",
						"ipv6_address_count",
						"1"),
				),
			},
		},
	})
}

func TestAccAWSInstance_ipv6AddressCountAndSingleAddressCausesError(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccInstanceConfigIpv6ErrorConfig,
				ExpectError: regexp.MustCompile("Only 1 of `ipv6_address_count` or `ipv6_addresses` can be specified"),
			},
		},
	})
}

func TestAccAWSInstance_ipv6_supportAddressCountWithIpv4(t *testing.T) {
	var v ec2.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceConfigIpv6SupportWithIpv4,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"aws_instance.foo", &v),
					resource.TestCheckResourceAttr(
						"aws_instance.foo",
						"ipv6_address_count",
						"1"),
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
			{
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
		PreCheck:        func() { testAccPreCheck(t) },
		IDRefreshName:   "aws_instance.foo_instance",
		IDRefreshIgnore: []string{"associate_public_ip_address"},
		Providers:       testAccProviders,
		CheckDestroy:    testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
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
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_instance.foo_instance",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
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
			{
				Config: testAccCheckInstanceConfigTags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testAccCheckTags(&v.Tags, "foo", "bar"),
					// Guard against regression of https://github.com/hashicorp/terraform/issues/914
					testAccCheckTags(&v.Tags, "#", ""),
				),
			},
			{
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

func TestAccAWSInstance_volumeTags(t *testing.T) {
	var v ec2.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckInstanceConfigNoVolumeTags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					resource.TestCheckNoResourceAttr(
						"aws_instance.foo", "volume_tags"),
				),
			},
			{
				Config: testAccCheckInstanceConfigWithVolumeTags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "volume_tags.%", "1"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "volume_tags.Name", "acceptance-test-volume-tag"),
				),
			},
			{
				Config: testAccCheckInstanceConfigWithVolumeTagsUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "volume_tags.%", "2"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "volume_tags.Name", "acceptance-test-volume-tag"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "volume_tags.Environment", "dev"),
				),
			},
			{
				Config: testAccCheckInstanceConfigNoVolumeTags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					resource.TestCheckNoResourceAttr(
						"aws_instance.foo", "volume_tags"),
				),
			},
		},
	})
}

func TestAccAWSInstance_volumeTagsComputed(t *testing.T) {
	var v ec2.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckInstanceConfigWithAttachedVolume,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
				),
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccAWSInstance_instanceProfileChange(t *testing.T) {
	var v ec2.Instance
	rName := acctest.RandString(5)

	testCheckInstanceProfile := func() resource.TestCheckFunc {
		return func(*terraform.State) error {
			if v.IamInstanceProfile == nil {
				return fmt.Errorf("Instance Profile is nil - we expected an InstanceProfile associated with the Instance")
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_instance.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceConfigWithoutInstanceProfile(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
				),
			},
			{
				Config: testAccInstanceConfigWithInstanceProfile(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testCheckInstanceProfile(),
				),
			},
		},
	})
}

func TestAccAWSInstance_withIamInstanceProfile(t *testing.T) {
	var v ec2.Instance
	rName := acctest.RandString(5)

	testCheckInstanceProfile := func() resource.TestCheckFunc {
		return func(*terraform.State) error {
			if v.IamInstanceProfile == nil {
				return fmt.Errorf("Instance Profile is nil - we expected an InstanceProfile associated with the Instance")
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_instance.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceConfigWithInstanceProfile(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testCheckInstanceProfile(),
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
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_instance.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
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
		PreCheck:        func() { testAccPreCheck(t) },
		IDRefreshName:   "aws_instance.foo",
		IDRefreshIgnore: []string{"associate_public_ip_address"},
		Providers:       testAccProviders,
		CheckDestroy:    testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
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

	keyPairName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:        func() { testAccPreCheck(t) },
		IDRefreshName:   "aws_instance.foo",
		IDRefreshIgnore: []string{"source_dest_check"},
		Providers:       testAccProviders,
		CheckDestroy:    testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceConfigKeyPair(keyPairName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testCheckKeyPair(keyPairName),
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
			{
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

// This test reproduces the bug here:
//   https://github.com/hashicorp/terraform/issues/1752
//
// I wish there were a way to exercise resources built with helper.Schema in a
// unit context, in which case this test could be moved there, but for now this
// will cover the bugfix.
//
// The following triggers "diffs didn't match during apply" without the fix in to
// set NewRemoved on the .# field when it changes to 0.
func TestAccAWSInstance_forceNewAndTagsDrift(t *testing.T) {
	var v ec2.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_instance.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceConfigForceNewAndTagsDrift,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					driftTags(&v),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccInstanceConfigForceNewAndTagsDrift_Update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
				),
			},
		},
	})
}

func TestAccAWSInstance_changeInstanceType(t *testing.T) {
	var before ec2.Instance
	var after ec2.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceConfigWithSmallInstanceType,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &before),
				),
			},
			{
				Config: testAccInstanceConfigUpdateInstanceType,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &after),
					testAccCheckInstanceNotRecreated(
						t, &before, &after),
				),
			},
		},
	})
}

func TestAccAWSInstance_primaryNetworkInterface(t *testing.T) {
	var instance ec2.Instance
	var ini ec2.NetworkInterface

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceConfigPrimaryNetworkInterface,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &instance),
					testAccCheckAWSENIExists("aws_network_interface.bar", &ini),
					resource.TestCheckResourceAttr("aws_instance.foo", "network_interface.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSInstance_primaryNetworkInterfaceSourceDestCheck(t *testing.T) {
	var instance ec2.Instance
	var ini ec2.NetworkInterface

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceConfigPrimaryNetworkInterfaceSourceDestCheck,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &instance),
					testAccCheckAWSENIExists("aws_network_interface.bar", &ini),
					resource.TestCheckResourceAttr("aws_instance.foo", "source_dest_check", "false"),
				),
			},
		},
	})
}

func TestAccAWSInstance_addSecondaryInterface(t *testing.T) {
	var before ec2.Instance
	var after ec2.Instance
	var iniPrimary ec2.NetworkInterface
	var iniSecondary ec2.NetworkInterface

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceConfigAddSecondaryNetworkInterfaceBefore,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &before),
					testAccCheckAWSENIExists("aws_network_interface.primary", &iniPrimary),
					resource.TestCheckResourceAttr("aws_instance.foo", "network_interface.#", "1"),
				),
			},
			{
				Config: testAccInstanceConfigAddSecondaryNetworkInterfaceAfter,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &after),
					testAccCheckAWSENIExists("aws_network_interface.secondary", &iniSecondary),
					resource.TestCheckResourceAttr("aws_instance.foo", "network_interface.#", "1"),
				),
			},
		},
	})
}

// https://github.com/hashicorp/terraform/issues/3205
func TestAccAWSInstance_addSecurityGroupNetworkInterface(t *testing.T) {
	var before ec2.Instance
	var after ec2.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceConfigAddSecurityGroupBefore,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &before),
					resource.TestCheckResourceAttr("aws_instance.foo", "vpc_security_group_ids.#", "1"),
				),
			},
			{
				Config: testAccInstanceConfigAddSecurityGroupAfter,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &after),
					resource.TestCheckResourceAttr("aws_instance.foo", "vpc_security_group_ids.#", "2"),
				),
			},
		},
	})
}

func testAccCheckInstanceNotRecreated(t *testing.T,
	before, after *ec2.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *before.InstanceId != *after.InstanceId {
			t.Fatalf("AWS Instance IDs have changed. Before %s. After %s", *before.InstanceId, *after.InstanceId)
		}
		return nil
	}
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
		resp, err := conn.DescribeInstances(&ec2.DescribeInstancesInput{
			InstanceIds: []*string{aws.String(rs.Primary.ID)},
		})
		if err == nil {
			for _, r := range resp.Reservations {
				for _, i := range r.Instances {
					if i.State != nil && *i.State.Name != "terminated" {
						return fmt.Errorf("Found unterminated instance: %s", i)
					}
				}
			}
		}

		// Verify the error is what we want
		if ae, ok := err.(awserr.Error); ok && ae.Code() == "InvalidInstanceID.NotFound" {
			continue
		}

		return err
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

func driftTags(instance *ec2.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		_, err := conn.CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{instance.InstanceId},
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Drift"),
					Value: aws.String("Happens"),
				},
			},
		})
		return err
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

const testAccInstanceConfigWithSmallInstanceType = `
resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-55a7ea65"
	availability_zone = "us-west-2a"

	instance_type = "m3.medium"

	tags {
	    Name = "tf-acctest"
	}
}
`

const testAccInstanceConfigUpdateInstanceType = `
resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-55a7ea65"
	availability_zone = "us-west-2a"

	instance_type = "m3.large"

	tags {
	    Name = "tf-acctest"
	}
}
`

const testAccInstanceGP2IopsDevice = `
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
	tags {
		Name = "testAccInstanceConfigSourceDestEnable"
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
}
`

const testAccInstanceConfigSourceDestDisable = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "testAccInstanceConfigSourceDestDisable"
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
	source_dest_check = false
}
`

func testAccInstanceConfigDisableAPITermination(val bool) string {
	return fmt.Sprintf(`
	resource "aws_vpc" "foo" {
		cidr_block = "10.1.0.0/16"
		tags {
			Name = "testAccInstanceConfigDisableAPITermination"
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
		disable_api_termination = %t
	}
	`, val)
}

const testAccInstanceConfigVPC = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "testAccInstanceConfigVPC"
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
`

const testAccInstanceConfigIpv6ErrorConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	assign_generated_ipv6_cidr_block = true
	tags {
		Name = "tf-ipv6-instance-acc-test"
	}
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
	ipv6_cidr_block = "${cidrsubnet(aws_vpc.foo.ipv6_cidr_block, 8, 1)}"
	tags {
		Name = "tf-ipv6-instance-acc-test"
	}
}

resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-c5eabbf5"
	instance_type = "t2.micro"
	subnet_id = "${aws_subnet.foo.id}"
	ipv6_addresses = ["2600:1f14:bb2:e501::10"]
	ipv6_address_count = 1
	tags {
		Name = "tf-ipv6-instance-acc-test"
	}
}
`

const testAccInstanceConfigIpv6Support = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	assign_generated_ipv6_cidr_block = true
	tags {
		Name = "tf-ipv6-instance-acc-test"
	}
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
	ipv6_cidr_block = "${cidrsubnet(aws_vpc.foo.ipv6_cidr_block, 8, 1)}"
	tags {
		Name = "tf-ipv6-instance-acc-test"
	}
}

resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-c5eabbf5"
	instance_type = "t2.micro"
	subnet_id = "${aws_subnet.foo.id}"

	ipv6_address_count = 1
	tags {
		Name = "tf-ipv6-instance-acc-test"
	}
}
`

const testAccInstanceConfigIpv6SupportWithIpv4 = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	assign_generated_ipv6_cidr_block = true
	tags {
		Name = "tf-ipv6-instance-acc-test"
	}
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
	ipv6_cidr_block = "${cidrsubnet(aws_vpc.foo.ipv6_cidr_block, 8, 1)}"
	tags {
		Name = "tf-ipv6-instance-acc-test"
	}
}

resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-c5eabbf5"
	instance_type = "t2.micro"
	subnet_id = "${aws_subnet.foo.id}"

	associate_public_ip_address = true
	ipv6_address_count = 1
	tags {
		Name = "tf-ipv6-instance-acc-test"
	}
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

const testAccCheckInstanceConfigWithAttachedVolume = `
data "aws_ami" "debian_jessie_latest" {
  most_recent = true

  filter {
    name   = "name"
    values = ["debian-jessie-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }

  filter {
    name   = "root-device-type"
    values = ["ebs"]
  }

  owners = ["379101102735"] # Debian
}

resource "aws_instance" "foo" {
  ami                         = "${data.aws_ami.debian_jessie_latest.id}"
  associate_public_ip_address = true
  count                       = 1
  instance_type               = "t2.medium"

  root_block_device {
    volume_size           = "10"
    volume_type           = "standard"
    delete_on_termination = true
  }

  tags {
    Name    = "test-terraform"
  }
}

resource "aws_ebs_volume" "test" {
  depends_on        = ["aws_instance.foo"]
  availability_zone = "${aws_instance.foo.availability_zone}"
  type       = "gp2"
  size              = "10"

  tags {
    Name = "test-terraform"
  }
}

resource "aws_volume_attachment" "test" {
  depends_on  = ["aws_ebs_volume.test"]
  device_name = "/dev/xvdg"
  volume_id   = "${aws_ebs_volume.test.id}"
  instance_id = "${aws_instance.foo.id}"
}
`

const testAccCheckInstanceConfigNoVolumeTags = `
resource "aws_instance" "foo" {
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

const testAccCheckInstanceConfigWithVolumeTags = `
resource "aws_instance" "foo" {
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

	ebs_block_device {
		device_name = "/dev/sdd"
		volume_size = 12
		encrypted = true
	}

	ephemeral_block_device {
		device_name = "/dev/sde"
		virtual_name = "ephemeral0"
	}

	volume_tags {
		Name = "acceptance-test-volume-tag"
	}
}
`

const testAccCheckInstanceConfigWithVolumeTagsUpdate = `
resource "aws_instance" "foo" {
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

	ebs_block_device {
		device_name = "/dev/sdd"
		volume_size = 12
		encrypted = true
	}

	ephemeral_block_device {
		device_name = "/dev/sde"
		virtual_name = "ephemeral0"
	}

	volume_tags {
		Name = "acceptance-test-volume-tag"
		Environment = "dev"
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

func testAccInstanceConfigWithoutInstanceProfile(rName string) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "test" {
	name = "test-%s"
	assume_role_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"Service\":[\"ec2.amazonaws.com\"]},\"Action\":[\"sts:AssumeRole\"]}]}"
}

resource "aws_iam_instance_profile" "test" {
	name = "test-%s"
	roles = ["${aws_iam_role.test.name}"]
}

resource "aws_instance" "foo" {
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
	tags {
		bar = "baz"
	}
}`, rName, rName)
}

func testAccInstanceConfigWithInstanceProfile(rName string) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "test" {
	name = "test-%s"
	assume_role_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"Service\":[\"ec2.amazonaws.com\"]},\"Action\":[\"sts:AssumeRole\"]}]}"
}

resource "aws_iam_instance_profile" "test" {
	name = "test-%s"
	roles = ["${aws_iam_role.test.name}"]
}

resource "aws_instance" "foo" {
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
	iam_instance_profile = "${aws_iam_instance_profile.test.name}"
	tags {
		bar = "baz"
	}
}`, rName, rName)
}

const testAccInstanceConfigPrivateIP = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "testAccInstanceConfigPrivateIP"
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
`

const testAccInstanceConfigAssociatePublicIPAndPrivateIP = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "testAccInstanceConfigAssociatePublicIPAndPrivateIP"
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
  vpc_security_group_ids = ["${aws_security_group.tf_test_foo.id}"]
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

func testAccInstanceConfigKeyPair(keyPairName string) string {
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
		Name = "testAccInstanceConfigKeyPair_TestAMI"
	}
}
`, keyPairName)
}

const testAccInstanceConfigRootBlockDeviceMismatch = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "testAccInstanceConfigRootBlockDeviceMismatch"
	}
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

const testAccInstanceConfigForceNewAndTagsDrift = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "testAccInstanceConfigForceNewAndTagsDrift"
	}
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_instance" "foo" {
	ami = "ami-22b9a343"
	instance_type = "t2.nano"
	subnet_id = "${aws_subnet.foo.id}"
}
`

const testAccInstanceConfigForceNewAndTagsDrift_Update = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "testAccInstanceConfigForceNewAndTagsDrift_Update"
	}
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_instance" "foo" {
	ami = "ami-22b9a343"
	instance_type = "t2.micro"
	subnet_id = "${aws_subnet.foo.id}"
}
`

const testAccInstanceConfigPrimaryNetworkInterface = `
resource "aws_vpc" "foo" {
  cidr_block = "172.16.0.0/16"
  tags {
    Name = "tf-instance-test"
  }
}

resource "aws_subnet" "foo" {
  vpc_id = "${aws_vpc.foo.id}"
  cidr_block = "172.16.10.0/24"
  availability_zone = "us-west-2a"
  tags {
    Name = "tf-instance-test"
  }
}

resource "aws_network_interface" "bar" {
  subnet_id = "${aws_subnet.foo.id}"
  private_ips = ["172.16.10.100"]
  tags {
    Name = "primary_network_interface"
  }
}

resource "aws_instance" "foo" {
	ami = "ami-22b9a343"
	instance_type = "t2.micro"
	network_interface {
	 network_interface_id = "${aws_network_interface.bar.id}"
	 device_index = 0
  }
}
`

const testAccInstanceConfigPrimaryNetworkInterfaceSourceDestCheck = `
resource "aws_vpc" "foo" {
  cidr_block = "172.16.0.0/16"
  tags {
    Name = "tf-instance-test"
  }
}

resource "aws_subnet" "foo" {
  vpc_id = "${aws_vpc.foo.id}"
  cidr_block = "172.16.10.0/24"
  availability_zone = "us-west-2a"
  tags {
    Name = "tf-instance-test"
  }
}

resource "aws_network_interface" "bar" {
  subnet_id = "${aws_subnet.foo.id}"
  private_ips = ["172.16.10.100"]
  source_dest_check = false
  tags {
    Name = "primary_network_interface"
  }
}

resource "aws_instance" "foo" {
	ami = "ami-22b9a343"
	instance_type = "t2.micro"
	network_interface {
	 network_interface_id = "${aws_network_interface.bar.id}"
	 device_index = 0
  }
}
`

const testAccInstanceConfigAddSecondaryNetworkInterfaceBefore = `
resource "aws_vpc" "foo" {
  cidr_block = "172.16.0.0/16"
  tags {
    Name = "tf-instance-test"
  }
}

resource "aws_subnet" "foo" {
  vpc_id = "${aws_vpc.foo.id}"
  cidr_block = "172.16.10.0/24"
  availability_zone = "us-west-2a"
  tags {
    Name = "tf-instance-test"
  }
}

resource "aws_network_interface" "primary" {
  subnet_id = "${aws_subnet.foo.id}"
  private_ips = ["172.16.10.100"]
  tags {
    Name = "primary_network_interface"
  }
}

resource "aws_network_interface" "secondary" {
  subnet_id = "${aws_subnet.foo.id}"
  private_ips = ["172.16.10.101"]
  tags {
    Name = "secondary_network_interface"
  }
}

resource "aws_instance" "foo" {
	ami = "ami-22b9a343"
	instance_type = "t2.micro"
	network_interface {
	 network_interface_id = "${aws_network_interface.primary.id}"
	 device_index = 0
  }
}
`

const testAccInstanceConfigAddSecondaryNetworkInterfaceAfter = `
resource "aws_vpc" "foo" {
  cidr_block = "172.16.0.0/16"
  tags {
    Name = "tf-instance-test"
  }
}

resource "aws_subnet" "foo" {
  vpc_id = "${aws_vpc.foo.id}"
  cidr_block = "172.16.10.0/24"
  availability_zone = "us-west-2a"
  tags {
    Name = "tf-instance-test"
  }
}

resource "aws_network_interface" "primary" {
  subnet_id = "${aws_subnet.foo.id}"
  private_ips = ["172.16.10.100"]
  tags {
    Name = "primary_network_interface"
  }
}

// Attach previously created network interface, observe no state diff on instance resource
resource "aws_network_interface" "secondary" {
  subnet_id = "${aws_subnet.foo.id}"
  private_ips = ["172.16.10.101"]
  tags {
    Name = "secondary_network_interface"
  }
  attachment {
    instance = "${aws_instance.foo.id}"
    device_index = 1
  }
}

resource "aws_instance" "foo" {
	ami = "ami-22b9a343"
	instance_type = "t2.micro"
	network_interface {
	 network_interface_id = "${aws_network_interface.primary.id}"
	 device_index = 0
  }
}
`

const testAccInstanceConfigAddSecurityGroupBefore = `
resource "aws_vpc" "foo" {
    cidr_block = "172.16.0.0/16"
        tags {
            Name = "tf-eni-test"
        }
}

resource "aws_subnet" "foo" {
    vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "172.16.10.0/24"
    availability_zone = "us-west-2a"
        tags {
            Name = "tf-foo-instance-add-sg-test"
        }
}

resource "aws_subnet" "bar" {
    vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "172.16.11.0/24"
    availability_zone = "us-west-2a"
        tags {
            Name = "tf-bar-instance-add-sg-test"
        }
}

resource "aws_security_group" "foo" {
  vpc_id = "${aws_vpc.foo.id}"
  description = "foo"
  name = "foo"
}

resource "aws_security_group" "bar" {
  vpc_id = "${aws_vpc.foo.id}"
  description = "bar"
  name = "bar"
}

resource "aws_instance" "foo" {
    ami = "ami-c5eabbf5"
    instance_type = "t2.micro"
    subnet_id = "${aws_subnet.bar.id}"
    associate_public_ip_address = false
    vpc_security_group_ids = [
      "${aws_security_group.foo.id}"
    ]
    tags {
        Name = "foo-instance-sg-add-test"
    }
}

resource "aws_network_interface" "bar" {
    subnet_id = "${aws_subnet.foo.id}"
    private_ips = ["172.16.10.100"]
    security_groups = ["${aws_security_group.foo.id}"]
    attachment {
        instance = "${aws_instance.foo.id}"
        device_index = 1
    }
    tags {
        Name = "bar_interface"
    }
}
`

const testAccInstanceConfigAddSecurityGroupAfter = `
resource "aws_vpc" "foo" {
    cidr_block = "172.16.0.0/16"
        tags {
            Name = "tf-eni-test"
        }
}

resource "aws_subnet" "foo" {
    vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "172.16.10.0/24"
    availability_zone = "us-west-2a"
        tags {
            Name = "tf-foo-instance-add-sg-test"
        }
}

resource "aws_subnet" "bar" {
    vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "172.16.11.0/24"
    availability_zone = "us-west-2a"
        tags {
            Name = "tf-bar-instance-add-sg-test"
        }
}

resource "aws_security_group" "foo" {
  vpc_id = "${aws_vpc.foo.id}"
  description = "foo"
  name = "foo"
}

resource "aws_security_group" "bar" {
  vpc_id = "${aws_vpc.foo.id}"
  description = "bar"
  name = "bar"
}

resource "aws_instance" "foo" {
    ami = "ami-c5eabbf5"
    instance_type = "t2.micro"
    subnet_id = "${aws_subnet.bar.id}"
    associate_public_ip_address = false
    vpc_security_group_ids = [
      "${aws_security_group.foo.id}",
      "${aws_security_group.bar.id}"
    ]
    tags {
        Name = "foo-instance-sg-add-test"
    }
}

resource "aws_network_interface" "bar" {
    subnet_id = "${aws_subnet.foo.id}"
    private_ips = ["172.16.10.100"]
    security_groups = ["${aws_security_group.foo.id}"]
    attachment {
        instance = "${aws_instance.foo.id}"
        device_index = 1
    }
    tags {
        Name = "bar_interface"
    }
}
`
