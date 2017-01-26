package spotinst

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/spotinst/spotinst-sdk-go/spotinst"
)

func TestAccSpotinstGroup_Basic(t *testing.T) {
	var group spotinst.AwsGroup
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSpotinstGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSpotinstGroupConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSpotinstGroupExists("spotinst_aws_group.foo", &group),
					testAccCheckSpotinstGroupAttributes(&group),
					resource.TestCheckResourceAttr("spotinst_aws_group.foo", "name", "terraform"),
					resource.TestCheckResourceAttr("spotinst_aws_group.foo", "description", "terraform"),
				),
			},
		},
	})
}

func TestAccSpotinstGroup_Updated(t *testing.T) {
	var group spotinst.AwsGroup
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSpotinstGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSpotinstGroupConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSpotinstGroupExists("spotinst_aws_group.foo", &group),
					testAccCheckSpotinstGroupAttributes(&group),
					resource.TestCheckResourceAttr("spotinst_aws_group.foo", "name", "terraform"),
					resource.TestCheckResourceAttr("spotinst_aws_group.foo", "description", "terraform"),
				),
			},
			{
				Config: testAccCheckSpotinstGroupConfigNewValue,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSpotinstGroupExists("spotinst_aws_group.foo", &group),
					testAccCheckSpotinstGroupAttributesUpdated(&group),
					resource.TestCheckResourceAttr("spotinst_aws_group.foo", "name", "terraform_updated"),
					resource.TestCheckResourceAttr("spotinst_aws_group.foo", "description", "terraform_updated"),
				),
			},
		},
	})
}

func testAccCheckSpotinstGroupDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*spotinst.Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "spotinst_aws_group" {
			continue
		}
		input := &spotinst.ReadAwsGroupInput{ID: spotinst.String(rs.Primary.ID)}
		resp, err := client.AwsGroupService.Read(input)
		if err == nil && resp != nil && resp.Group != nil {
			return fmt.Errorf("Group still exists")
		}
	}
	return nil
}

func testAccCheckSpotinstGroupAttributes(group *spotinst.AwsGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if spotinst.StringValue(group.Name) != "terraform" {
			return fmt.Errorf("Bad content: %v", group.Name)
		}
		return nil
	}
}

func testAccCheckSpotinstGroupAttributesUpdated(group *spotinst.AwsGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if spotinst.StringValue(group.Name) != "terraform_updated" {
			return fmt.Errorf("Bad content: %v", group.Name)
		}
		return nil
	}
}

func testAccCheckSpotinstGroupExists(n string, group *spotinst.AwsGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No resource ID is set")
		}
		client := testAccProvider.Meta().(*spotinst.Client)
		input := &spotinst.ReadAwsGroupInput{ID: spotinst.String(rs.Primary.ID)}
		resp, err := client.AwsGroupService.Read(input)
		if err != nil {
			return err
		}
		if spotinst.StringValue(resp.Group.Name) != rs.Primary.Attributes["name"] {
			return fmt.Errorf("Group not found: %+v,\n %+v\n", resp.Group, rs.Primary.Attributes)
		}
		*group = *resp.Group
		return nil
	}
}

const testAccCheckSpotinstGroupConfigBasic = `
resource "spotinst_aws_group" "foo" {
	name = "terraform"
	description = "terraform"
	product = "Linux/UNIX"

	capacity {
		target = 0
		minimum = 0
		maximum = 5
	}

	strategy {
		risk = 100
	}

	instance_types {
		ondemand = "c3.large"
		spot = ["c3.large", "m4.xlarge"]
	}

	availability_zone {
		name = "us-west-2b"
	}

	launch_specification {
		monitoring = false
		image_id = "ami-f0091d91"
		key_pair = "east"
		security_group_ids = ["default"]
	}

	scaling_up_policy {
        policy_name = "Scaling Policy 1"
        metric_name = "CPUUtilization"
        statistic = "average"
        unit = "percent"
        threshold = 80
        adjustment = 1
        namespace = "AWS/EC2"
        operator = "gte"
        period = 300
        evaluation_periods = 2
        cooldown = 300
        dimensions {
            env = "prod"
        }
    }
}`

const testAccCheckSpotinstGroupConfigNewValue = `
resource "spotinst_aws_group" "foo" {
	name = "terraform_updated"
	description = "terraform_updated"
	product = "Linux/UNIX"

	capacity {
		target = 0
		minimum = 0
		maximum = 5
	}

	strategy {
		risk = 100
	}

	instance_types {
		ondemand = "c3.large"
		spot = ["c3.large", "m4.xlarge"]
	}

	availability_zone {
		name = "us-west-2b"
	}

	launch_specification {
		monitoring = false
		image_id = "ami-f0091d91"
		key_pair = "east"
		security_group_ids = ["default"]
	}

	scaling_up_policy {
        policy_name = "Scaling Policy 2"
        metric_name = "CPUUtilization"
        statistic = "average"
        unit = "percent"
        threshold = 80
        adjustment = 1
        namespace = "AWS/EC2"
        operator = "gte"
        period = 300
        evaluation_periods = 2
        cooldown = 300
        dimensions {
            env = "dev"
        }
    }
}`
