package aws

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSElasticBeanstalkSolutionStackDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsElasticBeanstalkSolutionStackDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsElasticBeanstalkSolutionStackDataSourceID("data.aws_elastic_beanstalk_solution_stack.multi_docker"),
					resource.TestMatchResourceAttr("data.aws_elastic_beanstalk_solution_stack.multi_docker", "name", regexp.MustCompile("^64bit Amazon Linux (.*) Multi-container Docker (.*)$")),
				),
			},
		},
	})
}

func TestResourceValidateSolutionStackNameRegex(t *testing.T) {
	type testCases struct {
		Value    string
		ErrCount int
	}

	invalidCases := []testCases{
		{
			Value:    `\`,
			ErrCount: 1,
		},
		{
			Value:    `**`,
			ErrCount: 1,
		},
		{
			Value:    `(.+`,
			ErrCount: 1,
		},
	}

	for _, tc := range invalidCases {
		_, errors := validateSolutionStackNameRegex(tc.Value, "name_regex")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q to trigger a validation error.", tc.Value)
		}
	}

	validCases := []testCases{
		{
			Value:    `\/`,
			ErrCount: 0,
		},
		{
			Value:    `.*`,
			ErrCount: 0,
		},
		{
			Value:    `\b(?:\d{1,3}\.){3}\d{1,3}\b`,
			ErrCount: 0,
		},
	}

	for _, tc := range validCases {
		_, errors := validateSolutionStackNameRegex(tc.Value, "name_regex")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q not to trigger a validation error.", tc.Value)
		}
	}
}

func testAccCheckAwsElasticBeanstalkSolutionStackDataSourceDestroy(s *terraform.State) error {
	return nil
}

func testAccCheckAwsElasticBeanstalkSolutionStackDataSourceID(n string) resource.TestCheckFunc {
	// Wait for solution stacks
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find solution stack data source: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Solution stack data source ID not set")
		}
		return nil
	}
}

const testAccCheckAwsElasticBeanstalkSolutionStackDataSourceConfig = `
data "aws_elastic_beanstalk_solution_stack" "multi_docker" {
	most_recent = true
	name_regex  = "^64bit Amazon Linux (.*) Multi-container Docker (.*)$"
}
`
