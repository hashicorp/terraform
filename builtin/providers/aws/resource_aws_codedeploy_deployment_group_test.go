package aws

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/codedeploy"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCodeDeployDeploymentGroup_basic(t *testing.T) {
	var group codedeploy.DeploymentGroupInfo

	rName := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroup(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "app_name", "foo_app_"+rName),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "deployment_group_name", "foo_"+rName),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "deployment_config_name", "CodeDeployDefault.OneAtATime"),
					resource.TestMatchResourceAttr(
						"aws_codedeploy_deployment_group.foo", "service_role_arn",
						regexp.MustCompile("arn:aws:iam::[0-9]{12}:role/foo_role_.*")),

					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "ec2_tag_filter.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "ec2_tag_filter.2916377465.key", "filterkey"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "ec2_tag_filter.2916377465.type", "KEY_AND_VALUE"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "ec2_tag_filter.2916377465.value", "filtervalue"),

					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "alarm_configuration.#", "0"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "auto_rollback_configuration.#", "0"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "trigger_configuration.#", "0"),
				),
			},
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroupModified(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "app_name", "foo_app_"+rName),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "deployment_group_name", "bar_"+rName),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "deployment_config_name", "CodeDeployDefault.OneAtATime"),
					resource.TestMatchResourceAttr(
						"aws_codedeploy_deployment_group.foo", "service_role_arn",
						regexp.MustCompile("arn:aws:iam::[0-9]{12}:role/bar_role_.*")),

					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "ec2_tag_filter.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "ec2_tag_filter.2369538975.key", "filterkey"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "ec2_tag_filter.2369538975.type", "KEY_AND_VALUE"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "ec2_tag_filter.2369538975.value", "anotherfiltervalue"),

					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "alarm_configuration.#", "0"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "auto_rollback_configuration.#", "0"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "trigger_configuration.#", "0"),
				),
			},
		},
	})
}

func TestAccAWSCodeDeployDeploymentGroup_onPremiseTag(t *testing.T) {
	var group codedeploy.DeploymentGroupInfo

	rName := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroupOnPremiseTags(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "app_name", "foo_app_"+rName),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "deployment_group_name", "foo_"+rName),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "deployment_config_name", "CodeDeployDefault.OneAtATime"),

					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "on_premises_instance_tag_filter.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "on_premises_instance_tag_filter.2916377465.key", "filterkey"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "on_premises_instance_tag_filter.2916377465.type", "KEY_AND_VALUE"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "on_premises_instance_tag_filter.2916377465.value", "filtervalue"),
				),
			},
		},
	})
}

func TestAccAWSCodeDeployDeploymentGroup_disappears(t *testing.T) {
	var group codedeploy.DeploymentGroupInfo
	rName := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroup(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo", &group),
					testAccAWSCodeDeployDeploymentGroupDisappears(&group),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAWSCodeDeployDeploymentGroup_triggerConfiguration_basic(t *testing.T) {
	var group codedeploy.DeploymentGroupInfo

	rName := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_create(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "app_name", "foo-app-"+rName),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "deployment_group_name", "foo-group-"+rName),
					testAccCheckTriggerEvents(&group, "foo-trigger", []string{
						"DeploymentFailure",
					}),
				),
			},
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_update(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "app_name", "foo-app-"+rName),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "deployment_group_name", "foo-group-"+rName),
					testAccCheckTriggerEvents(&group, "foo-trigger", []string{
						"DeploymentFailure",
						"DeploymentSuccess",
					}),
				),
			},
		},
	})
}

func TestAccAWSCodeDeployDeploymentGroup_triggerConfiguration_multiple(t *testing.T) {
	var group codedeploy.DeploymentGroupInfo

	rName := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_createMultiple(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "app_name", "foo-app-"+rName),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "deployment_group_name", "foo-group-"+rName),
					testAccCheckTriggerEvents(&group, "foo-trigger", []string{
						"DeploymentFailure",
					}),
					testAccCheckTriggerEvents(&group, "bar-trigger", []string{
						"InstanceFailure",
					}),
					testAccCheckTriggerTargetArn(&group, "bar-trigger",
						regexp.MustCompile("^arn:aws:sns:[^:]+:[0-9]{12}:bar-topic-"+rName+"$")),
				),
			},
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_updateMultiple(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "app_name", "foo-app-"+rName),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "deployment_group_name", "foo-group-"+rName),
					testAccCheckTriggerEvents(&group, "foo-trigger", []string{
						"DeploymentFailure",
						"DeploymentStart",
						"DeploymentStop",
						"DeploymentSuccess",
					}),
					testAccCheckTriggerEvents(&group, "bar-trigger", []string{
						"InstanceFailure",
					}),
					testAccCheckTriggerTargetArn(&group, "bar-trigger",
						regexp.MustCompile("^arn:aws:sns:[^:]+:[0-9]{12}:baz-topic-"+rName+"$")),
				),
			},
		},
	})
}

func TestAccAWSCodeDeployDeploymentGroup_autoRollbackConfiguration_create(t *testing.T) {
	var group codedeploy.DeploymentGroupInfo

	rName := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: test_config_auto_rollback_configuration_delete(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.#", "0"),
				),
			},
			resource.TestStep{
				Config: test_config_auto_rollback_configuration_create(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.0.enabled", "true"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.0.events.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.0.events.135881253", "DEPLOYMENT_FAILURE"),
				),
			},
		},
	})
}

func TestAccAWSCodeDeployDeploymentGroup_autoRollbackConfiguration_update(t *testing.T) {
	var group codedeploy.DeploymentGroupInfo

	rName := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: test_config_auto_rollback_configuration_create(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.0.enabled", "true"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.0.events.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.0.events.135881253", "DEPLOYMENT_FAILURE"),
				),
			},
			resource.TestStep{
				Config: test_config_auto_rollback_configuration_update(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.0.enabled", "true"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.0.events.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.0.events.104943466", "DEPLOYMENT_STOP_ON_ALARM"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.0.events.135881253", "DEPLOYMENT_FAILURE"),
				),
			},
		},
	})
}

func TestAccAWSCodeDeployDeploymentGroup_autoRollbackConfiguration_delete(t *testing.T) {
	var group codedeploy.DeploymentGroupInfo

	rName := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: test_config_auto_rollback_configuration_create(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.0.enabled", "true"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.0.events.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.0.events.135881253", "DEPLOYMENT_FAILURE"),
				),
			},
			resource.TestStep{
				Config: test_config_auto_rollback_configuration_delete(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.#", "0"),
				),
			},
		},
	})
}

func TestAccAWSCodeDeployDeploymentGroup_autoRollbackConfiguration_disable(t *testing.T) {
	var group codedeploy.DeploymentGroupInfo

	rName := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: test_config_auto_rollback_configuration_create(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.0.enabled", "true"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.0.events.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.0.events.135881253", "DEPLOYMENT_FAILURE"),
				),
			},
			resource.TestStep{
				Config: test_config_auto_rollback_configuration_disable(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.0.enabled", "false"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.0.events.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "auto_rollback_configuration.0.events.135881253", "DEPLOYMENT_FAILURE"),
				),
			},
		},
	})
}

func TestAccAWSCodeDeployDeploymentGroup_alarmConfiguration_create(t *testing.T) {
	var group codedeploy.DeploymentGroupInfo

	rName := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: test_config_alarm_configuration_delete(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.#", "0"),
				),
			},
			resource.TestStep{
				Config: test_config_alarm_configuration_create(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.enabled", "true"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.alarms.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.alarms.2356372769", "foo"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.ignore_poll_alarm_failure", "false"),
				),
			},
		},
	})
}

func TestAccAWSCodeDeployDeploymentGroup_alarmConfiguration_update(t *testing.T) {
	var group codedeploy.DeploymentGroupInfo

	rName := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: test_config_alarm_configuration_create(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.enabled", "true"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.alarms.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.alarms.2356372769", "foo"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.ignore_poll_alarm_failure", "false"),
				),
			},
			resource.TestStep{
				Config: test_config_alarm_configuration_update(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.enabled", "true"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.alarms.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.alarms.1996459178", "bar"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.alarms.2356372769", "foo"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.ignore_poll_alarm_failure", "true"),
				),
			},
		},
	})
}

func TestAccAWSCodeDeployDeploymentGroup_alarmConfiguration_delete(t *testing.T) {
	var group codedeploy.DeploymentGroupInfo

	rName := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: test_config_alarm_configuration_create(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.enabled", "true"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.alarms.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.alarms.2356372769", "foo"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.ignore_poll_alarm_failure", "false"),
				),
			},
			resource.TestStep{
				Config: test_config_alarm_configuration_delete(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.#", "0"),
				),
			},
		},
	})
}

func TestAccAWSCodeDeployDeploymentGroup_alarmConfiguration_disable(t *testing.T) {
	var group codedeploy.DeploymentGroupInfo

	rName := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: test_config_alarm_configuration_create(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.enabled", "true"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.alarms.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.alarms.2356372769", "foo"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.ignore_poll_alarm_failure", "false"),
				),
			},
			resource.TestStep{
				Config: test_config_alarm_configuration_disable(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.enabled", "false"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.alarms.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.alarms.2356372769", "foo"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "alarm_configuration.0.ignore_poll_alarm_failure", "false"),
				),
			},
		},
	})
}

func TestValidateAWSCodeDeployTriggerEvent(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "DeploymentStart",
			ErrCount: 0,
		},
		{
			Value:    "DeploymentStop",
			ErrCount: 0,
		},
		{
			Value:    "DeploymentSuccess",
			ErrCount: 0,
		},
		{
			Value:    "DeploymentFailure",
			ErrCount: 0,
		},
		{
			Value:    "DeploymentRollback",
			ErrCount: 0,
		},
		{
			Value:    "InstanceStart",
			ErrCount: 0,
		},
		{
			Value:    "InstanceSuccess",
			ErrCount: 0,
		},
		{
			Value:    "InstanceFailure",
			ErrCount: 0,
		},
		{
			Value:    "DeploymentStarts",
			ErrCount: 1,
		},
		{
			Value:    "InstanceFail",
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
		_, errors := validateTriggerEvent(tc.Value, "trigger_event")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Trigger event validation failed for event type %q: %q", tc.Value, errors)
		}
	}
}

func TestBuildTriggerConfigs(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"trigger_events": schema.NewSet(schema.HashString, []interface{}{
				"DeploymentFailure",
			}),
			"trigger_name":       "foo-trigger",
			"trigger_target_arn": "arn:aws:sns:us-west-2:123456789012:foo-topic",
		},
	}

	expected := []*codedeploy.TriggerConfig{
		&codedeploy.TriggerConfig{
			TriggerEvents: []*string{
				aws.String("DeploymentFailure"),
			},
			TriggerName:      aws.String("foo-trigger"),
			TriggerTargetArn: aws.String("arn:aws:sns:us-west-2:123456789012:foo-topic"),
		},
	}

	actual := buildTriggerConfigs(input)

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("buildTriggerConfigs output is not correct.\nGot:\n%#v\nExpected:\n%#v\n",
			actual, expected)
	}
}

func TestTriggerConfigsToMap(t *testing.T) {
	input := []*codedeploy.TriggerConfig{
		&codedeploy.TriggerConfig{
			TriggerEvents: []*string{
				aws.String("DeploymentFailure"),
				aws.String("InstanceFailure"),
			},
			TriggerName:      aws.String("bar-trigger"),
			TriggerTargetArn: aws.String("arn:aws:sns:us-west-2:123456789012:bar-topic"),
		},
	}

	expected := map[string]interface{}{
		"trigger_events": schema.NewSet(schema.HashString, []interface{}{
			"DeploymentFailure",
			"InstanceFailure",
		}),
		"trigger_name":       "bar-trigger",
		"trigger_target_arn": "arn:aws:sns:us-west-2:123456789012:bar-topic",
	}

	actual := triggerConfigsToMap(input)[0]

	fatal := false

	if actual["trigger_name"] != expected["trigger_name"] {
		fatal = true
	}

	if actual["trigger_target_arn"] != expected["trigger_target_arn"] {
		fatal = true
	}

	actualEvents := actual["trigger_events"].(*schema.Set)
	expectedEvents := expected["trigger_events"].(*schema.Set)
	if !actualEvents.Equal(expectedEvents) {
		fatal = true
	}

	if fatal {
		t.Fatalf("triggerConfigsToMap output is not correct.\nGot:\n%#v\nExpected:\n%#v\n",
			actual, expected)
	}
}

func TestBuildAutoRollbackConfig(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"events": schema.NewSet(schema.HashString, []interface{}{
				"DEPLOYMENT_FAILURE",
			}),
			"enabled": true,
		},
	}

	expected := &codedeploy.AutoRollbackConfiguration{
		Events: []*string{
			aws.String("DEPLOYMENT_FAILURE"),
		},
		Enabled: aws.Bool(true),
	}

	actual := buildAutoRollbackConfig(input)

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("buildAutoRollbackConfig output is not correct.\nGot:\n%#v\nExpected:\n%#v\n",
			actual, expected)
	}
}

func TestAutoRollbackConfigToMap(t *testing.T) {
	input := &codedeploy.AutoRollbackConfiguration{
		Events: []*string{
			aws.String("DEPLOYMENT_FAILURE"),
			aws.String("DEPLOYMENT_STOP_ON_ALARM"),
		},
		Enabled: aws.Bool(false),
	}

	expected := map[string]interface{}{
		"events": schema.NewSet(schema.HashString, []interface{}{
			"DEPLOYMENT_FAILURE",
			"DEPLOYMENT_STOP_ON_ALARM",
		}),
		"enabled": false,
	}

	actual := autoRollbackConfigToMap(input)[0]

	fatal := false

	if actual["enabled"] != expected["enabled"] {
		fatal = true
	}

	actualEvents := actual["events"].(*schema.Set)
	expectedEvents := expected["events"].(*schema.Set)
	if !actualEvents.Equal(expectedEvents) {
		fatal = true
	}

	if fatal {
		t.Fatalf("autoRollbackConfigToMap output is not correct.\nGot:\n%#v\nExpected:\n%#v\n",
			actual, expected)
	}
}

func TestBuildAlarmConfig(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"alarms": schema.NewSet(schema.HashString, []interface{}{
				"foo-alarm",
			}),
			"enabled":                   true,
			"ignore_poll_alarm_failure": false,
		},
	}

	expected := &codedeploy.AlarmConfiguration{
		Alarms: []*codedeploy.Alarm{
			{
				Name: aws.String("foo-alarm"),
			},
		},
		Enabled:                aws.Bool(true),
		IgnorePollAlarmFailure: aws.Bool(false),
	}

	actual := buildAlarmConfig(input)

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("buildAlarmConfig output is not correct.\nGot:\n%#v\nExpected:\n%#v\n",
			actual, expected)
	}
}

func TestAlarmConfigToMap(t *testing.T) {
	input := &codedeploy.AlarmConfiguration{
		Alarms: []*codedeploy.Alarm{
			{
				Name: aws.String("bar-alarm"),
			},
			{
				Name: aws.String("foo-alarm"),
			},
		},
		Enabled:                aws.Bool(false),
		IgnorePollAlarmFailure: aws.Bool(true),
	}

	expected := map[string]interface{}{
		"alarms": schema.NewSet(schema.HashString, []interface{}{
			"bar-alarm",
			"foo-alarm",
		}),
		"enabled":                   false,
		"ignore_poll_alarm_failure": true,
	}

	actual := alarmConfigToMap(input)[0]

	fatal := false

	if actual["enabled"] != expected["enabled"] {
		fatal = true
	}

	if actual["ignore_poll_alarm_failure"] != expected["ignore_poll_alarm_failure"] {
		fatal = true
	}

	actualAlarms := actual["alarms"].(*schema.Set)
	expectedAlarms := expected["alarms"].(*schema.Set)
	if !actualAlarms.Equal(expectedAlarms) {
		fatal = true
	}

	if fatal {
		t.Fatalf("alarmConfigToMap output is not correct.\nGot:\n%#v\nExpected:\n%#v\n",
			actual, expected)
	}
}

func testAccCheckTriggerEvents(group *codedeploy.DeploymentGroupInfo, triggerName string, expectedEvents []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		for _, actual := range group.TriggerConfigurations {
			if *actual.TriggerName == triggerName {

				numberOfEvents := len(actual.TriggerEvents)
				if numberOfEvents != len(expectedEvents) {
					return fmt.Errorf("Trigger events do not match. Expected: %d. Got: %d.",
						len(expectedEvents), numberOfEvents)
				}

				actualEvents := make([]string, 0, numberOfEvents)
				for _, event := range actual.TriggerEvents {
					actualEvents = append(actualEvents, *event)
				}
				sort.Strings(actualEvents)

				if !reflect.DeepEqual(actualEvents, expectedEvents) {
					return fmt.Errorf("Trigger events do not match.\nExpected: %v\nGot: %v\n",
						expectedEvents, actualEvents)
				}
				break
			}
		}
		return nil
	}
}

func testAccCheckTriggerTargetArn(group *codedeploy.DeploymentGroupInfo, triggerName string, r *regexp.Regexp) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, actual := range group.TriggerConfigurations {
			if *actual.TriggerName == triggerName {
				if !r.MatchString(*actual.TriggerTargetArn) {
					return fmt.Errorf("Trigger target arn does not match regular expression.\nRegex: %v\nTriggerTargetArn: %v\n",
						r, *actual.TriggerTargetArn)
				}
				break
			}
		}
		return nil
	}
}

func testAccCheckAWSCodeDeployDeploymentGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).codedeployconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_codedeploy_deployment_group" {
			continue
		}

		resp, err := conn.GetDeploymentGroup(&codedeploy.GetDeploymentGroupInput{
			ApplicationName:     aws.String(rs.Primary.Attributes["app_name"]),
			DeploymentGroupName: aws.String(rs.Primary.Attributes["deployment_group_name"]),
		})

		if ae, ok := err.(awserr.Error); ok && ae.Code() == "ApplicationDoesNotExistException" {
			continue
		}

		if err == nil {
			if resp.DeploymentGroupInfo.DeploymentGroupName != nil {
				return fmt.Errorf("CodeDeploy deployment group still exists:\n%#v", *resp.DeploymentGroupInfo.DeploymentGroupName)
			}
		}

		return err
	}

	return nil
}

func testAccAWSCodeDeployDeploymentGroupDisappears(group *codedeploy.DeploymentGroupInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).codedeployconn
		opts := &codedeploy.DeleteDeploymentGroupInput{
			ApplicationName:     group.ApplicationName,
			DeploymentGroupName: group.DeploymentGroupName,
		}
		if _, err := conn.DeleteDeploymentGroup(opts); err != nil {
			return err
		}
		return resource.Retry(40*time.Minute, func() *resource.RetryError {
			opts := &codedeploy.GetDeploymentGroupInput{
				ApplicationName:     group.ApplicationName,
				DeploymentGroupName: group.DeploymentGroupName,
			}
			_, err := conn.GetDeploymentGroup(opts)
			if err != nil {
				codedeploy, ok := err.(awserr.Error)
				if ok && codedeploy.Code() == "DeploymentGroupDoesNotExistException" {
					return nil
				}
				return resource.NonRetryableError(
					fmt.Errorf("Error retrieving CodeDeploy Deployment Group: %s", err))
			}
			return resource.RetryableError(fmt.Errorf(
				"Waiting for CodeDeploy Deployment Group: %v", group.DeploymentGroupName))
		})
	}
}

func testAccCheckAWSCodeDeployDeploymentGroupExists(name string, group *codedeploy.DeploymentGroupInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		conn := testAccProvider.Meta().(*AWSClient).codedeployconn

		resp, err := conn.GetDeploymentGroup(&codedeploy.GetDeploymentGroupInput{
			ApplicationName:     aws.String(rs.Primary.Attributes["app_name"]),
			DeploymentGroupName: aws.String(rs.Primary.Attributes["deployment_group_name"]),
		})

		if err != nil {
			return err
		}

		*group = *resp.DeploymentGroupInfo

		return nil
	}
}

func testAccAWSCodeDeployDeploymentGroup(rName string) string {
	return fmt.Sprintf(`
resource "aws_codedeploy_app" "foo_app" {
	name = "foo_app_%s"
}

resource "aws_iam_role_policy" "foo_policy" {
	name = "foo_policy_%s"
	role = "${aws_iam_role.foo_role.id}"
	policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Effect": "Allow",
			"Action": [
				"autoscaling:CompleteLifecycleAction",
				"autoscaling:DeleteLifecycleHook",
				"autoscaling:DescribeAutoScalingGroups",
				"autoscaling:DescribeLifecycleHooks",
				"autoscaling:PutLifecycleHook",
				"autoscaling:RecordLifecycleActionHeartbeat",
				"ec2:DescribeInstances",
				"ec2:DescribeInstanceStatus",
				"tag:GetTags",
				"tag:GetResources"
			],
			"Resource": "*"
		}
	]
}
EOF
}

resource "aws_iam_role" "foo_role" {
	name = "foo_role_%s"
	assume_role_policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Sid": "",
			"Effect": "Allow",
			"Principal": {
				"Service": [
					"codedeploy.amazonaws.com"
				]
			},
			"Action": "sts:AssumeRole"
		}
	]
}
EOF
}

resource "aws_codedeploy_deployment_group" "foo" {
	app_name = "${aws_codedeploy_app.foo_app.name}"
	deployment_group_name = "foo_%s"
	service_role_arn = "${aws_iam_role.foo_role.arn}"
	ec2_tag_filter {
		key = "filterkey"
		type = "KEY_AND_VALUE"
		value = "filtervalue"
	}
}`, rName, rName, rName, rName)
}

func testAccAWSCodeDeployDeploymentGroupModified(rName string) string {
	return fmt.Sprintf(`
resource "aws_codedeploy_app" "foo_app" {
	name = "foo_app_%s"
}

resource "aws_iam_role_policy" "foo_policy" {
	name = "foo_policy_%s"
	role = "${aws_iam_role.bar_role.id}"
	policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Effect": "Allow",
			"Action": [
				"autoscaling:CompleteLifecycleAction",
				"autoscaling:DeleteLifecycleHook",
				"autoscaling:DescribeAutoScalingGroups",
				"autoscaling:DescribeLifecycleHooks",
				"autoscaling:PutLifecycleHook",
				"autoscaling:RecordLifecycleActionHeartbeat",
				"ec2:DescribeInstances",
				"ec2:DescribeInstanceStatus",
				"tag:GetTags",
				"tag:GetResources"
			],
			"Resource": "*"
		}
	]
}
EOF
}

resource "aws_iam_role" "bar_role" {
	name = "bar_role_%s"
	assume_role_policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Sid": "",
			"Effect": "Allow",
			"Principal": {
				"Service": [
					"codedeploy.amazonaws.com"
				]
			},
			"Action": "sts:AssumeRole"
		}
	]
}
EOF
}

resource "aws_codedeploy_deployment_group" "foo" {
	app_name = "${aws_codedeploy_app.foo_app.name}"
	deployment_group_name = "bar_%s"
	service_role_arn = "${aws_iam_role.bar_role.arn}"
	ec2_tag_filter {
		key = "filterkey"
		type = "KEY_AND_VALUE"
		value = "anotherfiltervalue"
	}
}`, rName, rName, rName, rName)
}

func testAccAWSCodeDeployDeploymentGroupOnPremiseTags(rName string) string {
	return fmt.Sprintf(`
resource "aws_codedeploy_app" "foo_app" {
	name = "foo_app_%s"
}

resource "aws_iam_role_policy" "foo_policy" {
	name = "foo_policy_%s"
	role = "${aws_iam_role.foo_role.id}"
	policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Effect": "Allow",
			"Action": [
				"autoscaling:CompleteLifecycleAction",
				"autoscaling:DeleteLifecycleHook",
				"autoscaling:DescribeAutoScalingGroups",
				"autoscaling:DescribeLifecycleHooks",
				"autoscaling:PutLifecycleHook",
				"autoscaling:RecordLifecycleActionHeartbeat",
				"ec2:DescribeInstances",
				"ec2:DescribeInstanceStatus",
				"tag:GetTags",
				"tag:GetResources"
			],
			"Resource": "*"
		}
	]
}
EOF
}

resource "aws_iam_role" "foo_role" {
	name = "foo_role_%s"
	assume_role_policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Sid": "",
			"Effect": "Allow",
			"Principal": {
				"Service": [
					"codedeploy.amazonaws.com"
				]
			},
			"Action": "sts:AssumeRole"
		}
	]
}
EOF
}

resource "aws_codedeploy_deployment_group" "foo" {
	app_name = "${aws_codedeploy_app.foo_app.name}"
	deployment_group_name = "foo_%s"
	service_role_arn = "${aws_iam_role.foo_role.arn}"
	on_premises_instance_tag_filter {
		key = "filterkey"
		type = "KEY_AND_VALUE"
		value = "filtervalue"
	}
}`, rName, rName, rName, rName)
}

func baseCodeDeployConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_codedeploy_app" "foo_app" {
	name = "foo-app-%s"
}

resource "aws_iam_role_policy" "foo_policy" {
	name = "foo-policy-%s"
	role = "${aws_iam_role.foo_role.id}"
	policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Effect": "Allow",
			"Action": [
				"autoscaling:CompleteLifecycleAction",
				"autoscaling:DeleteLifecycleHook",
				"autoscaling:DescribeAutoScalingGroups",
				"autoscaling:DescribeLifecycleHooks",
				"autoscaling:PutLifecycleHook",
				"autoscaling:RecordLifecycleActionHeartbeat",
				"codedeploy:*",
				"ec2:DescribeInstances",
				"ec2:DescribeInstanceStatus",
				"tag:GetTags",
				"tag:GetResources",
				"sns:Publish"
			],
			"Resource": "*"
		}
	]
}
EOF
}

resource "aws_iam_role" "foo_role" {
	name = "foo-role-%s"
	assume_role_policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Sid": "",
			"Effect": "Allow",
			"Principal": {
				"Service": "codedeploy.amazonaws.com"
			},
			"Action": "sts:AssumeRole"
		}
	]
}
EOF
}

resource "aws_sns_topic" "foo_topic" {
	name = "foo-topic-%s"
}`, rName, rName, rName, rName)
}

func testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_create(rName string) string {
	return fmt.Sprintf(`

	%s

resource "aws_codedeploy_deployment_group" "foo_group" {
	app_name = "${aws_codedeploy_app.foo_app.name}"
	deployment_group_name = "foo-group-%s"
	service_role_arn = "${aws_iam_role.foo_role.arn}"

	trigger_configuration {
		trigger_events = ["DeploymentFailure"]
		trigger_name = "foo-trigger"
		trigger_target_arn = "${aws_sns_topic.foo_topic.arn}"
	}
}`, baseCodeDeployConfig(rName), rName)
}

func testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_update(rName string) string {
	return fmt.Sprintf(`

	%s

resource "aws_codedeploy_deployment_group" "foo_group" {
	app_name = "${aws_codedeploy_app.foo_app.name}"
	deployment_group_name = "foo-group-%s"
	service_role_arn = "${aws_iam_role.foo_role.arn}"

	trigger_configuration {
		trigger_events = ["DeploymentSuccess", "DeploymentFailure"]
		trigger_name = "foo-trigger"
		trigger_target_arn = "${aws_sns_topic.foo_topic.arn}"
	}
}`, baseCodeDeployConfig(rName), rName)
}

func testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_createMultiple(rName string) string {
	return fmt.Sprintf(`

	%s

resource "aws_sns_topic" "bar_topic" {
	name = "bar-topic-%s"
}

resource "aws_codedeploy_deployment_group" "foo_group" {
	app_name = "${aws_codedeploy_app.foo_app.name}"
	deployment_group_name = "foo-group-%s"
	service_role_arn = "${aws_iam_role.foo_role.arn}"

	trigger_configuration {
		trigger_events = ["DeploymentFailure"]
		trigger_name = "foo-trigger"
		trigger_target_arn = "${aws_sns_topic.foo_topic.arn}"
	}

	trigger_configuration {
		trigger_events = ["InstanceFailure"]
		trigger_name = "bar-trigger"
		trigger_target_arn = "${aws_sns_topic.bar_topic.arn}"
	}
}`, baseCodeDeployConfig(rName), rName, rName)
}

func testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_updateMultiple(rName string) string {
	return fmt.Sprintf(`

	%s

resource "aws_sns_topic" "bar_topic" {
	name = "bar-topic-%s"
}

resource "aws_sns_topic" "baz_topic" {
	name = "baz-topic-%s"
}

resource "aws_codedeploy_deployment_group" "foo_group" {
	app_name = "${aws_codedeploy_app.foo_app.name}"
	deployment_group_name = "foo-group-%s"
	service_role_arn = "${aws_iam_role.foo_role.arn}"

	trigger_configuration {
		trigger_events = ["DeploymentStart", "DeploymentSuccess", "DeploymentFailure", "DeploymentStop"]
		trigger_name = "foo-trigger"
		trigger_target_arn = "${aws_sns_topic.foo_topic.arn}"
	}

	trigger_configuration {
		trigger_events = ["InstanceFailure"]
		trigger_name = "bar-trigger"
		trigger_target_arn = "${aws_sns_topic.baz_topic.arn}"
	}
}`, baseCodeDeployConfig(rName), rName, rName, rName)
}

func test_config_auto_rollback_configuration_create(rName string) string {
	return fmt.Sprintf(`

  %s

resource "aws_codedeploy_deployment_group" "foo_group" {
  app_name = "${aws_codedeploy_app.foo_app.name}"
  deployment_group_name = "foo-group-%s"
  service_role_arn = "${aws_iam_role.foo_role.arn}"

  auto_rollback_configuration {
    enabled = true
    events = ["DEPLOYMENT_FAILURE"]
  }
}`, baseCodeDeployConfig(rName), rName)
}

func test_config_auto_rollback_configuration_update(rName string) string {
	return fmt.Sprintf(`

  %s

resource "aws_codedeploy_deployment_group" "foo_group" {
  app_name = "${aws_codedeploy_app.foo_app.name}"
  deployment_group_name = "foo-group-%s"
  service_role_arn = "${aws_iam_role.foo_role.arn}"

  auto_rollback_configuration {
    enabled = true
    events = ["DEPLOYMENT_FAILURE", "DEPLOYMENT_STOP_ON_ALARM"]
  }
}`, baseCodeDeployConfig(rName), rName)
}

func test_config_auto_rollback_configuration_delete(rName string) string {
	return fmt.Sprintf(`

  %s

resource "aws_codedeploy_deployment_group" "foo_group" {
  app_name = "${aws_codedeploy_app.foo_app.name}"
  deployment_group_name = "foo-group-%s"
  service_role_arn = "${aws_iam_role.foo_role.arn}"
}`, baseCodeDeployConfig(rName), rName)
}

func test_config_auto_rollback_configuration_disable(rName string) string {
	return fmt.Sprintf(`

  %s

resource "aws_codedeploy_deployment_group" "foo_group" {
  app_name = "${aws_codedeploy_app.foo_app.name}"
  deployment_group_name = "foo-group-%s"
  service_role_arn = "${aws_iam_role.foo_role.arn}"

  auto_rollback_configuration {
    enabled = false
    events = ["DEPLOYMENT_FAILURE"]
  }
}`, baseCodeDeployConfig(rName), rName)
}

func test_config_alarm_configuration_create(rName string) string {
	return fmt.Sprintf(`

  %s

resource "aws_codedeploy_deployment_group" "foo_group" {
  app_name = "${aws_codedeploy_app.foo_app.name}"
  deployment_group_name = "foo-group-%s"
  service_role_arn = "${aws_iam_role.foo_role.arn}"

  alarm_configuration {
    alarms = ["foo"]
    enabled = true
  }
}`, baseCodeDeployConfig(rName), rName)
}

func test_config_alarm_configuration_update(rName string) string {
	return fmt.Sprintf(`

  %s

resource "aws_codedeploy_deployment_group" "foo_group" {
  app_name = "${aws_codedeploy_app.foo_app.name}"
  deployment_group_name = "foo-group-%s"
  service_role_arn = "${aws_iam_role.foo_role.arn}"

  alarm_configuration {
    alarms = ["foo", "bar"]
    enabled = true
    ignore_poll_alarm_failure = true
  }
}`, baseCodeDeployConfig(rName), rName)
}

func test_config_alarm_configuration_delete(rName string) string {
	return fmt.Sprintf(`

  %s

resource "aws_codedeploy_deployment_group" "foo_group" {
  app_name = "${aws_codedeploy_app.foo_app.name}"
  deployment_group_name = "foo-group-%s"
  service_role_arn = "${aws_iam_role.foo_role.arn}"
}`, baseCodeDeployConfig(rName), rName)
}

func test_config_alarm_configuration_disable(rName string) string {
	return fmt.Sprintf(`

  %s

resource "aws_codedeploy_deployment_group" "foo_group" {
  app_name = "${aws_codedeploy_app.foo_app.name}"
  deployment_group_name = "foo-group-%s"
  service_role_arn = "${aws_iam_role.foo_role.arn}"

  alarm_configuration {
    alarms = ["foo"]
    enabled = false
  }
}`, baseCodeDeployConfig(rName), rName)
}
