package pagerduty

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccPagerDutyService_import(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.com", username)
	escalationPolicy := fmt.Sprintf("tf-%s", acctest.RandString(5))
	service := fmt.Sprintf("tf-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyServiceConfig(username, email, escalationPolicy, service),
			},

			{
				ResourceName:      "pagerduty_service.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPagerDutyServiceWithIncidentUrgency_import(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.com", username)
	escalationPolicy := fmt.Sprintf("tf-%s", acctest.RandString(5))
	service := fmt.Sprintf("tf-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyServiceWithIncidentUrgencyRulesConfig(username, email, escalationPolicy, service),
			},

			{
				ResourceName:      "pagerduty_service.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
