package aws

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSLightsailInstance_basic(t *testing.T) {
	var conf lightsail.Instance
	lightsailName := fmt.Sprintf("tf-test-lightsail-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_lightsail_instance.lightsail_instance_test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSLightsailInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLightsailInstanceConfig_basic(lightsailName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSLightsailInstanceExists("aws_lightsail_instance.lightsail_instance_test", &conf),
					resource.TestCheckResourceAttrSet("aws_lightsail_instance.lightsail_instance_test", "availability_zone"),
					resource.TestCheckResourceAttrSet("aws_lightsail_instance.lightsail_instance_test", "blueprint_id"),
					resource.TestCheckResourceAttrSet("aws_lightsail_instance.lightsail_instance_test", "bundle_id"),
					resource.TestCheckResourceAttrSet("aws_lightsail_instance.lightsail_instance_test", "key_pair_name"),
				),
			},
		},
	})
}

func TestAccAWSLightsailInstance_euRegion(t *testing.T) {
	var conf lightsail.Instance
	lightsailName := fmt.Sprintf("tf-test-lightsail-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_lightsail_instance.lightsail_instance_test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSLightsailInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLightsailInstanceConfig_euRegion(lightsailName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSLightsailInstanceExists("aws_lightsail_instance.lightsail_instance_test", &conf),
					resource.TestCheckResourceAttrSet("aws_lightsail_instance.lightsail_instance_test", "availability_zone"),
					resource.TestCheckResourceAttrSet("aws_lightsail_instance.lightsail_instance_test", "blueprint_id"),
					resource.TestCheckResourceAttrSet("aws_lightsail_instance.lightsail_instance_test", "bundle_id"),
					resource.TestCheckResourceAttrSet("aws_lightsail_instance.lightsail_instance_test", "key_pair_name"),
				),
			},
		},
	})
}

func TestAccAWSLightsailInstance_disapear(t *testing.T) {
	var conf lightsail.Instance
	lightsailName := fmt.Sprintf("tf-test-lightsail-%d", acctest.RandInt())

	testDestroy := func(*terraform.State) error {
		// reach out and DELETE the Instance
		conn := testAccProvider.Meta().(*AWSClient).lightsailconn
		_, err := conn.DeleteInstance(&lightsail.DeleteInstanceInput{
			InstanceName: aws.String(lightsailName),
		})

		if err != nil {
			return fmt.Errorf("Error deleting Lightsail Instance in disapear test")
		}

		// sleep 7 seconds to give it time, so we don't have to poll
		time.Sleep(7 * time.Second)

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLightsailInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLightsailInstanceConfig_basic(lightsailName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSLightsailInstanceExists("aws_lightsail_instance.lightsail_instance_test", &conf),
					testDestroy,
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckAWSLightsailInstanceExists(n string, res *lightsail.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("No LightsailInstance ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).lightsailconn

		respInstance, err := conn.GetInstance(&lightsail.GetInstanceInput{
			InstanceName: aws.String(rs.Primary.Attributes["name"]),
		})

		if err != nil {
			return err
		}

		if respInstance == nil || respInstance.Instance == nil {
			return fmt.Errorf("Instance (%s) not found", rs.Primary.Attributes["name"])
		}
		*res = *respInstance.Instance
		return nil
	}
}

func testAccCheckAWSLightsailInstanceDestroy(s *terraform.State) error {

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_lightsail_instance" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).lightsailconn

		respInstance, err := conn.GetInstance(&lightsail.GetInstanceInput{
			InstanceName: aws.String(rs.Primary.Attributes["name"]),
		})

		if err == nil {
			if respInstance.Instance != nil {
				return fmt.Errorf("LightsailInstance %q still exists", rs.Primary.ID)
			}
		}

		// Verify the error
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NotFoundException" {
				return nil
			}
		}
		return err
	}

	return nil
}

func testAccAWSLightsailInstanceConfig_basic(lightsailName string) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-east-1"
}
resource "aws_lightsail_instance" "lightsail_instance_test" {
  name              = "%s"
  availability_zone = "us-east-1b"
  blueprint_id      = "gitlab_8_12_6"
  bundle_id         = "nano_1_0"
}
`, lightsailName)
}

func testAccAWSLightsailInstanceConfig_euRegion(lightsailName string) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "eu-west-1"
}
resource "aws_lightsail_instance" "lightsail_instance_test" {
  name              = "%s"
  availability_zone = "eu-west-1a"
  blueprint_id = "joomla_3_6_5"
  bundle_id         = "nano_1_0"
}
`, lightsailName)
}
