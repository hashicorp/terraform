package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSecurityGroup_importBasic(t *testing.T) {
	checkFn := func(s []*terraform.InstanceState) error {
		// Expect 3: group, 2 rules
		if len(s) != 3 {
			return fmt.Errorf("expected 3 states: %#v", s)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfig,
			},

			{
				ResourceName:     "aws_security_group.web",
				ImportState:      true,
				ImportStateCheck: checkFn,
			},
		},
	})
}

func TestAccAWSSecurityGroup_importIpv6(t *testing.T) {
	checkFn := func(s []*terraform.InstanceState) error {
		// Expect 3: group, 2 rules
		if len(s) != 3 {
			return fmt.Errorf("expected 3 states: %#v", s)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfigIpv6,
			},

			{
				ResourceName:     "aws_security_group.web",
				ImportState:      true,
				ImportStateCheck: checkFn,
			},
		},
	})
}

func TestAccAWSSecurityGroup_importSelf(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfig_importSelf,
			},

			{
				ResourceName:      "aws_security_group.allow_all",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSSecurityGroup_importSourceSecurityGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfig_importSourceSecurityGroup,
			},

			{
				ResourceName:      "aws_security_group.test_group_1",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSSecurityGroup_importIPRangeAndSecurityGroupWithSameRules(t *testing.T) {
	checkFn := func(s []*terraform.InstanceState) error {
		// Expect 4: group, 3 rules
		if len(s) != 4 {
			return fmt.Errorf("expected 4 states: %#v", s)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfig_importIPRangeAndSecurityGroupWithSameRules,
			},

			{
				ResourceName:     "aws_security_group.test_group_1",
				ImportState:      true,
				ImportStateCheck: checkFn,
			},
		},
	})
}

func TestAccAWSSecurityGroup_importIPRangesWithSameRules(t *testing.T) {
	checkFn := func(s []*terraform.InstanceState) error {
		// Expect 4: group, 2 rules
		if len(s) != 3 {
			return fmt.Errorf("expected 3 states: %#v", s)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfig_importIPRangesWithSameRules,
			},

			{
				ResourceName:     "aws_security_group.test_group_1",
				ImportState:      true,
				ImportStateCheck: checkFn,
			},
		},
	})
}

func TestAccAWSSecurityGroup_importPrefixList(t *testing.T) {
	checkFn := func(s []*terraform.InstanceState) error {
		// Expect 2: group, 1 rule
		if len(s) != 2 {
			return fmt.Errorf("expected 2 states: %#v", s)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSecurityGroupConfigPrefixListEgress,
			},

			{
				ResourceName:     "aws_security_group.egress",
				ImportState:      true,
				ImportStateCheck: checkFn,
			},
		},
	})
}
