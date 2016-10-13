package aws

import (
	"encoding/base64"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSpotFleetRequest_changePriceForcesNewRequest(t *testing.T) {
	var before, after ec2.SpotFleetRequestConfig
	rName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSpotFleetRequestDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSpotFleetRequestConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSSpotFleetRequestExists(
						"aws_spot_fleet_request.foo", &before),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "spot_request_state", "active"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "spot_price", "0.005"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.#", "1"),
				),
			},
			resource.TestStep{
				Config: testAccAWSSpotFleetRequestConfigChangeSpotBidPrice(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSSpotFleetRequestExists(
						"aws_spot_fleet_request.foo", &after),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "spot_request_state", "active"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "spot_price", "0.01"),
					testAccCheckAWSSpotFleetRequestConfigRecreated(t, &before, &after),
				),
			},
		},
	})
}

func TestAccAWSSpotFleetRequest_lowestPriceAzOrSubnetInRegion(t *testing.T) {
	var sfr ec2.SpotFleetRequestConfig
	rName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSpotFleetRequestDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSpotFleetRequestConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSSpotFleetRequestExists(
						"aws_spot_fleet_request.foo", &sfr),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "spot_request_state", "active"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSSpotFleetRequest_lowestPriceAzInGivenList(t *testing.T) {
	var sfr ec2.SpotFleetRequestConfig
	rName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSpotFleetRequestDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSpotFleetRequestConfigWithAzs(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSSpotFleetRequestExists(
						"aws_spot_fleet_request.foo", &sfr),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "spot_request_state", "active"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.1590006269.availability_zone", "us-west-2a"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.3809475891.availability_zone", "us-west-2b"),
				),
			},
		},
	})
}

func TestAccAWSSpotFleetRequest_lowestPriceSubnetInGivenList(t *testing.T) {
	var sfr ec2.SpotFleetRequestConfig
	rName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSpotFleetRequestDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSpotFleetRequestConfigWithSubnet(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSSpotFleetRequestExists(
						"aws_spot_fleet_request.foo", &sfr),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "spot_request_state", "active"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.#", "2"),
				),
			},
		},
	})
}

func TestAccAWSSpotFleetRequest_multipleInstanceTypesInSameAz(t *testing.T) {
	var sfr ec2.SpotFleetRequestConfig
	rName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSpotFleetRequestDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSpotFleetRequestConfigMultipleInstanceTypesinSameAz(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSSpotFleetRequestExists(
						"aws_spot_fleet_request.foo", &sfr),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "spot_request_state", "active"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.1590006269.instance_type", "m1.small"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.1590006269.availability_zone", "us-west-2a"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.3079734941.instance_type", "m3.large"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.3079734941.availability_zone", "us-west-2a"),
				),
			},
		},
	})
}

func TestAccAWSSpotFleetRequest_multipleInstanceTypesInSameSubnet(t *testing.T) {
	var sfr ec2.SpotFleetRequestConfig
	rName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSpotFleetRequestDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSpotFleetRequestConfigMultipleInstanceTypesinSameSubnet(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSSpotFleetRequestExists(
						"aws_spot_fleet_request.foo", &sfr),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "spot_request_state", "active"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.#", "2"),
				),
			},
		},
	})
}

func TestAccAWSSpotFleetRequest_overriddingSpotPrice(t *testing.T) {
	var sfr ec2.SpotFleetRequestConfig
	rName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSpotFleetRequestDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSpotFleetRequestConfigOverridingSpotPrice(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSSpotFleetRequestExists(
						"aws_spot_fleet_request.foo", &sfr),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "spot_request_state", "active"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "spot_price", "0.005"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.522395050.spot_price", "0.01"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.522395050.instance_type", "m3.large"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.1590006269.spot_price", ""), //there will not be a value here since it's not overriding
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.1590006269.instance_type", "m1.small"),
				),
			},
		},
	})
}

func TestAccAWSSpotFleetRequest_diversifiedAllocation(t *testing.T) {
	var sfr ec2.SpotFleetRequestConfig
	rName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSpotFleetRequestDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSpotFleetRequestConfigDiversifiedAllocation(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSSpotFleetRequestExists(
						"aws_spot_fleet_request.foo", &sfr),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "spot_request_state", "active"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.#", "3"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "allocation_strategy", "diversified"),
				),
			},
		},
	})
}

func TestAccAWSSpotFleetRequest_withWeightedCapacity(t *testing.T) {
	var sfr ec2.SpotFleetRequestConfig
	rName := acctest.RandString(10)

	fulfillSleep := func() resource.TestCheckFunc {
		// sleep so that EC2 can fuflill the request. We do this to guard against a
		// regression and possible leak where we'll destroy the request and the
		// associated IAM role before anything is actually provisioned and running,
		// thus leaking when those newly started instances are attempted to be
		// destroyed
		// See https://github.com/hashicorp/terraform/pull/8938
		return func(s *terraform.State) error {
			log.Printf("[DEBUG] Test: Sleep to allow EC2 to actually begin fulfilling TestAccAWSSpotFleetRequest_withWeightedCapacity request")
			time.Sleep(1 * time.Minute)
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSpotFleetRequestDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSpotFleetRequestConfigWithWeightedCapacity(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					fulfillSleep(),
					testAccCheckAWSSpotFleetRequestExists(
						"aws_spot_fleet_request.foo", &sfr),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "spot_request_state", "active"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.2325690000.weighted_capacity", "3"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.2325690000.instance_type", "r3.large"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.3079734941.weighted_capacity", "6"),
					resource.TestCheckResourceAttr(
						"aws_spot_fleet_request.foo", "launch_specification.3079734941.instance_type", "m3.large"),
				),
			},
		},
	})
}

func TestAccAWSSpotFleetRequest_CannotUseEmptyKeyName(t *testing.T) {
	_, errors := validateSpotFleetRequestKeyName("", "key_name")
	if len(errors) == 0 {
		t.Fatalf("Expected the key name to trigger a validation error")
	}
}

func testAccCheckAWSSpotFleetRequestConfigRecreated(t *testing.T,
	before, after *ec2.SpotFleetRequestConfig) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if before.SpotFleetRequestId == after.SpotFleetRequestId {
			t.Fatalf("Expected change of Spot Fleet Request IDs, but both were %v", before.SpotFleetRequestId)
		}
		return nil
	}
}

func testAccCheckAWSSpotFleetRequestExists(
	n string, sfr *ec2.SpotFleetRequestConfig) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Spot fleet request with that id exists")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		params := &ec2.DescribeSpotFleetRequestsInput{
			SpotFleetRequestIds: []*string{&rs.Primary.ID},
		}
		resp, err := conn.DescribeSpotFleetRequests(params)

		if err != nil {
			return err
		}

		if v := len(resp.SpotFleetRequestConfigs); v != 1 {
			return fmt.Errorf("Expected 1 request returned, got %d", v)
		}

		*sfr = *resp.SpotFleetRequestConfigs[0]

		return nil
	}
}

func testAccCheckAWSSpotFleetRequest_LaunchSpecAttributes(
	sfr *ec2.SpotFleetRequestConfig) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(sfr.SpotFleetRequestConfig.LaunchSpecifications) == 0 {
			return fmt.Errorf("Missing launch specification")
		}

		spec := *sfr.SpotFleetRequestConfig.LaunchSpecifications[0]

		if *spec.InstanceType != "m1.small" {
			return fmt.Errorf("Unexpected launch specification instance type: %s", *spec.InstanceType)
		}

		if *spec.ImageId != "ami-d06a90b0" {
			return fmt.Errorf("Unexpected launch specification image id: %s", *spec.ImageId)
		}

		if *spec.SpotPrice != "0.01" {
			return fmt.Errorf("Unexpected launch specification spot price: %s", *spec.SpotPrice)
		}

		if *spec.WeightedCapacity != 2 {
			return fmt.Errorf("Unexpected launch specification weighted capacity: %f", *spec.WeightedCapacity)
		}

		if *spec.UserData != base64.StdEncoding.EncodeToString([]byte("hello-world")) {
			return fmt.Errorf("Unexpected launch specification user data: %s", *spec.UserData)
		}

		return nil
	}
}

func testAccCheckAWSSpotFleetRequestDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_spot_fleet_request" {
			continue
		}

		_, err := conn.CancelSpotFleetRequests(&ec2.CancelSpotFleetRequestsInput{
			SpotFleetRequestIds: []*string{aws.String(rs.Primary.ID)},
			TerminateInstances:  aws.Bool(true),
		})

		if err != nil {
			return fmt.Errorf("Error cancelling spot request (%s): %s", rs.Primary.ID, err)
		}
	}

	return nil
}

func testAccAWSSpotFleetRequestConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_key_pair" "debugging" {
	key_name = "tmp-key-%s"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}

resource "aws_iam_policy_attachment" "test-attach" {
    name = "test-attachment"
    roles = ["${aws_iam_role.test-role.name}"]
    policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2SpotFleetRole"
}

resource "aws_iam_role" "test-role" {
    name = "test-role-%s"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "spotfleet.amazonaws.com",
          "ec2.amazonaws.com"
        ]
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_spot_fleet_request" "foo" {
    iam_fleet_role = "${aws_iam_role.test-role.arn}"
    spot_price = "0.005"
    target_capacity = 2
    valid_until = "2019-11-04T20:44:20Z"
    terminate_instances_with_expiration = true
    launch_specification {
        instance_type = "m1.small"
        ami = "ami-d06a90b0"
        key_name = "${aws_key_pair.debugging.key_name}"
    }
    depends_on = ["aws_iam_policy_attachment.test-attach"]
}
`, rName, rName)
}

func testAccAWSSpotFleetRequestConfigChangeSpotBidPrice(rName string) string {
	return fmt.Sprintf(`
resource "aws_key_pair" "debugging" {
	key_name = "tmp-key-%s"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}

resource "aws_iam_policy_attachment" "test-attach" {
    name = "test-attachment"
    roles = ["${aws_iam_role.test-role.name}"]
    policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2SpotFleetRole"
}

resource "aws_iam_role" "test-role" {
    name = "test-role-%s"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "spotfleet.amazonaws.com",
          "ec2.amazonaws.com"
        ]
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_spot_fleet_request" "foo" {
    iam_fleet_role = "${aws_iam_role.test-role.arn}"
    spot_price = "0.01"
    target_capacity = 2
    valid_until = "2019-11-04T20:44:20Z"
    terminate_instances_with_expiration = true
    launch_specification {
        instance_type = "m1.small"
        ami = "ami-d06a90b0"
        key_name = "${aws_key_pair.debugging.key_name}"
    }
    depends_on = ["aws_iam_policy_attachment.test-attach"]
}
`, rName, rName)
}

func testAccAWSSpotFleetRequestConfigWithAzs(rName string) string {
	return fmt.Sprintf(`
resource "aws_key_pair" "debugging" {
	key_name = "tmp-key-%s"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}

resource "aws_iam_policy_attachment" "test-attach" {
    name = "test-attachment"
    roles = ["${aws_iam_role.test-role.name}"]
    policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2SpotFleetRole"
}

resource "aws_iam_role" "test-role" {
    name = "test-role-%s"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "spotfleet.amazonaws.com",
          "ec2.amazonaws.com"
        ]
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_spot_fleet_request" "foo" {
    iam_fleet_role = "${aws_iam_role.test-role.arn}"
    spot_price = "0.005"
    target_capacity = 2
    valid_until = "2019-11-04T20:44:20Z"
    terminate_instances_with_expiration = true
    launch_specification {
        instance_type = "m1.small"
        ami = "ami-d06a90b0"
        key_name = "${aws_key_pair.debugging.key_name}"
	availability_zone = "us-west-2a"
    }
    launch_specification {
        instance_type = "m1.small"
        ami = "ami-d06a90b0"
        key_name = "${aws_key_pair.debugging.key_name}"
	availability_zone = "us-west-2b"
    }
    depends_on = ["aws_iam_policy_attachment.test-attach"]
}
`, rName, rName)
}

func testAccAWSSpotFleetRequestConfigWithSubnet(rName string) string {
	return fmt.Sprintf(`
resource "aws_key_pair" "debugging" {
	key_name = "tmp-key-%s"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}

resource "aws_iam_policy_attachment" "test-attach" {
    name = "test-attachment"
    roles = ["${aws_iam_role.test-role.name}"]
    policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2SpotFleetRole"
}

resource "aws_iam_role" "test-role" {
    name = "test-role-%s"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "spotfleet.amazonaws.com",
          "ec2.amazonaws.com"
        ]
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_vpc" "foo" {
    cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo" {
    cidr_block = "10.1.1.0/24"
    vpc_id = "${aws_vpc.foo.id}"
    availability_zone = "us-west-2a"
}

resource "aws_subnet" "bar" {
    cidr_block = "10.1.20.0/24"
    vpc_id = "${aws_vpc.foo.id}"
    availability_zone = "us-west-2b"
}

resource "aws_spot_fleet_request" "foo" {
    iam_fleet_role = "${aws_iam_role.test-role.arn}"
    spot_price = "0.005"
    target_capacity = 4
    valid_until = "2019-11-04T20:44:20Z"
    terminate_instances_with_expiration = true
    launch_specification {
        instance_type = "m3.large"
        ami = "ami-d0f506b0"
        key_name = "${aws_key_pair.debugging.key_name}"
	subnet_id = "${aws_subnet.foo.id}"
    }
    launch_specification {
        instance_type = "m3.large"
        ami = "ami-d0f506b0"
        key_name = "${aws_key_pair.debugging.key_name}"
	subnet_id = "${aws_subnet.bar.id}"
    }
    depends_on = ["aws_iam_policy_attachment.test-attach"]
}
`, rName, rName)
}

func testAccAWSSpotFleetRequestConfigMultipleInstanceTypesinSameAz(rName string) string {
	return fmt.Sprintf(`
resource "aws_key_pair" "debugging" {
	key_name = "tmp-key-%s"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}

resource "aws_iam_policy_attachment" "test-attach" {
    name = "test-attachment"
    roles = ["${aws_iam_role.test-role.name}"]
    policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2SpotFleetRole"
}

resource "aws_iam_role" "test-role" {
    name = "test-role-%s"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "spotfleet.amazonaws.com",
          "ec2.amazonaws.com"
        ]
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_spot_fleet_request" "foo" {
    iam_fleet_role = "${aws_iam_role.test-role.arn}"
    spot_price = "0.005"
    target_capacity = 2
    valid_until = "2019-11-04T20:44:20Z"
    terminate_instances_with_expiration = true
    launch_specification {
        instance_type = "m1.small"
        ami = "ami-d06a90b0"
        key_name = "${aws_key_pair.debugging.key_name}"
        availability_zone = "us-west-2a"
    }
    launch_specification {
        instance_type = "m3.large"
        ami = "ami-d06a90b0"
        key_name = "${aws_key_pair.debugging.key_name}"
        availability_zone = "us-west-2a"
    }
    depends_on = ["aws_iam_policy_attachment.test-attach"]
}
`, rName, rName)
}

func testAccAWSSpotFleetRequestConfigMultipleInstanceTypesinSameSubnet(rName string) string {
	return fmt.Sprintf(`
resource "aws_key_pair" "debugging" {
	key_name = "tmp-key-%s"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}

resource "aws_iam_policy_attachment" "test-attach" {
    name = "test-attachment"
    roles = ["${aws_iam_role.test-role.name}"]
    policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2SpotFleetRole"
}

resource "aws_iam_role" "test-role" {
    name = "test-role-%s"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "spotfleet.amazonaws.com",
          "ec2.amazonaws.com"
        ]
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_vpc" "foo" {
    cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo" {
    cidr_block = "10.1.1.0/24"
    vpc_id = "${aws_vpc.foo.id}"
    availability_zone = "us-west-2a"
}

resource "aws_spot_fleet_request" "foo" {
    iam_fleet_role = "${aws_iam_role.test-role.arn}"
    spot_price = "0.005"
    target_capacity = 4
    valid_until = "2019-11-04T20:44:20Z"
    terminate_instances_with_expiration = true
    launch_specification {
        instance_type = "m3.large"
        ami = "ami-d0f506b0"
        key_name = "${aws_key_pair.debugging.key_name}"
	subnet_id = "${aws_subnet.foo.id}"
    }
    launch_specification {
        instance_type = "r3.large"
        ami = "ami-d0f506b0"
        key_name = "${aws_key_pair.debugging.key_name}"
	subnet_id = "${aws_subnet.foo.id}"
    }
    depends_on = ["aws_iam_policy_attachment.test-attach"]
}
`, rName, rName)
}

func testAccAWSSpotFleetRequestConfigOverridingSpotPrice(rName string) string {
	return fmt.Sprintf(`
resource "aws_key_pair" "debugging" {
	key_name = "tmp-key-%s"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}

resource "aws_iam_policy_attachment" "test-attach" {
    name = "test-attachment"
    roles = ["${aws_iam_role.test-role.name}"]
    policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2SpotFleetRole"
}

resource "aws_iam_role" "test-role" {
    name = "test-role-%s"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "spotfleet.amazonaws.com",
          "ec2.amazonaws.com"
        ]
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_spot_fleet_request" "foo" {
    iam_fleet_role = "${aws_iam_role.test-role.arn}"
    spot_price = "0.005"
    target_capacity = 2
    valid_until = "2019-11-04T20:44:20Z"
    terminate_instances_with_expiration = true
    launch_specification {
        instance_type = "m1.small"
        ami = "ami-d06a90b0"
        key_name = "${aws_key_pair.debugging.key_name}"
        availability_zone = "us-west-2a"
    }
    launch_specification {
        instance_type = "m3.large"
        ami = "ami-d06a90b0"
        key_name = "${aws_key_pair.debugging.key_name}"
        availability_zone = "us-west-2a"
        spot_price = "0.01"
    }
    depends_on = ["aws_iam_policy_attachment.test-attach"]
}
`, rName, rName)
}

func testAccAWSSpotFleetRequestConfigDiversifiedAllocation(rName string) string {
	return fmt.Sprintf(`
resource "aws_key_pair" "debugging" {
	key_name = "tmp-key-%s"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}

resource "aws_iam_policy_attachment" "test-attach" {
    name = "test-attachment"
    roles = ["${aws_iam_role.test-role.name}"]
    policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2SpotFleetRole"
}

resource "aws_iam_role" "test-role" {
    name = "test-role-%s"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "spotfleet.amazonaws.com",
          "ec2.amazonaws.com"
        ]
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_spot_fleet_request" "foo" {
    iam_fleet_role = "${aws_iam_role.test-role.arn}"
    spot_price = "0.7"
    target_capacity = 30
    valid_until = "2019-11-04T20:44:20Z"
    allocation_strategy = "diversified"
    terminate_instances_with_expiration = true
    launch_specification {
        instance_type = "m1.small"
        ami = "ami-d06a90b0"
        key_name = "${aws_key_pair.debugging.key_name}"
        availability_zone = "us-west-2a"
    }
    launch_specification {
        instance_type = "m3.large"
        ami = "ami-d06a90b0"
        key_name = "${aws_key_pair.debugging.key_name}"
        availability_zone = "us-west-2a"
    }
    launch_specification {
        instance_type = "r3.large"
        ami = "ami-d06a90b0"
        key_name = "${aws_key_pair.debugging.key_name}"
        availability_zone = "us-west-2a"
    }
    depends_on = ["aws_iam_policy_attachment.test-attach"]
}
`, rName, rName)
}

func testAccAWSSpotFleetRequestConfigWithWeightedCapacity(rName string) string {
	return fmt.Sprintf(`
resource "aws_key_pair" "debugging" {
	key_name = "tmp-key-%s"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 phodgson@thoughtworks.com"
}

resource "aws_iam_policy_attachment" "test-attach" {
    name = "test-attachment"
    roles = ["${aws_iam_role.test-role.name}"]
    policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2SpotFleetRole"
}

resource "aws_iam_role" "test-role" {
    name = "test-role-%s"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "spotfleet.amazonaws.com",
          "ec2.amazonaws.com"
        ]
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_spot_fleet_request" "foo" {
    iam_fleet_role = "${aws_iam_role.test-role.arn}"
    spot_price = "0.7"
    target_capacity = 10
    valid_until = "2019-11-04T20:44:20Z"
    terminate_instances_with_expiration = true
    launch_specification {
        instance_type = "m3.large"
        ami = "ami-d06a90b0"
        key_name = "${aws_key_pair.debugging.key_name}"
        availability_zone = "us-west-2a"
        weighted_capacity = "6"
    }
    launch_specification {
        instance_type = "r3.large"
        ami = "ami-d06a90b0"
        key_name = "${aws_key_pair.debugging.key_name}"
        availability_zone = "us-west-2a"
        weighted_capacity = "3"
    }
    depends_on = ["aws_iam_policy_attachment.test-attach"]
}
`, rName, rName)
}
