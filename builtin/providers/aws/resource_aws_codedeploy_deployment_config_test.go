package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/codedeploy"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCodeDeployDeploymentConfig_fleetPercent(t *testing.T) {
	var config codedeploy.DeploymentConfigInfo

	rName := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCodeDeployDeploymentConfigFleet(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentConfigExists("aws_codedeploy_deployment_config.foo", &config),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_config.foo", "minimum_healthy_hosts.0.type", "FLEET_PERCENT"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_config.foo", "minimum_healthy_hosts.0.value", "75"),
				),
			},
		},
	})
}

func TestAccAWSCodeDeployDeploymentConfig_hostCount(t *testing.T) {
	var config codedeploy.DeploymentConfigInfo

	rName := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCodeDeployDeploymentConfigHostCount(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentConfigExists("aws_codedeploy_deployment_config.foo", &config),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_config.foo", "minimum_healthy_hosts.0.type", "HOST_COUNT"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_config.foo", "minimum_healthy_hosts.0.value", "1"),
				),
			},
		},
	})
}

func TestValidateAWSCodeDeployMinimumHealthyHostsType(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "FLEET_PERCENT",
			ErrCount: 0,
		},
		{
			Value:    "HOST_COUNT",
			ErrCount: 0,
		},
		{
			Value:    "host_count",
			ErrCount: 1,
		},
		{
			Value:    "hostcount",
			ErrCount: 1,
		},
		{
			Value:    "FleetPercent",
			ErrCount: 1,
		},
		{
			Value:    "Foo",
			ErrCount: 1,
		},
		{
			Value:    "",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateMinimumHealtyHostsType(tc.Value, "minimum_healthy_hosts_type")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Minimum Healthy Hosts validation failed for type %q: %q", tc.Value, errors)
		}
	}
}

func testAccCheckAWSCodeDeployDeploymentConfigDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).codedeployconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_codedeploy_deployment_config" {
			continue
		}

		resp, err := conn.GetDeploymentConfig(&codedeploy.GetDeploymentConfigInput{
			DeploymentConfigName: aws.String(rs.Primary.ID),
		})

		if ae, ok := err.(awserr.Error); ok && ae.Code() == "DeploymentConfigDoesNotExistException" {
			continue
		}

		if err == nil {
			if resp.DeploymentConfigInfo != nil {
				return fmt.Errorf("CodeDeploy deployment config still exists:\n%#v", *resp.DeploymentConfigInfo.DeploymentConfigName)
			}
		}

		return err
	}

	return nil
}

func testAccCheckAWSCodeDeployDeploymentConfigExists(name string, config *codedeploy.DeploymentConfigInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		conn := testAccProvider.Meta().(*AWSClient).codedeployconn

		resp, err := conn.GetDeploymentConfig(&codedeploy.GetDeploymentConfigInput{
			DeploymentConfigName: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		*config = *resp.DeploymentConfigInfo

		return nil
	}
}

func testAccAWSCodeDeployDeploymentConfigFleet(rName string) string {
	return fmt.Sprintf(`
resource "aws_codedeploy_deployment_config" "foo" {
	deployment_config_name = "test-deployment-config-%s"
	minimum_healthy_hosts {
		type = "FLEET_PERCENT"
		value = 75
	}
}`, rName)
}

func testAccAWSCodeDeployDeploymentConfigHostCount(rName string) string {
	return fmt.Sprintf(`
resource "aws_codedeploy_deployment_config" "foo" {
	deployment_config_name = "test-deployment-config-%s"
	minimum_healthy_hosts {
		type = "HOST_COUNT"
		value = 1
	}
}`, rName)
}
