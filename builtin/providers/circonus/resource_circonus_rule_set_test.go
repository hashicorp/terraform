package circonus

import (
	"fmt"
	"strings"
	"testing"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccCirconusRuleSet_basic(t *testing.T) {
	checkName := fmt.Sprintf("ICMP Ping check - %s", acctest.RandString(5))
	contactGroupName := fmt.Sprintf("ops-staging-sev3 - %s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusRuleSet,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCirconusRuleSetConfigFmt, contactGroupName, checkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("circonus_rule_set.icmp-latency-alarm", "check"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "metric_name", "maximum"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "metric_type", "numeric"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "notes", "Simple check to create notifications based on ICMP performance."),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "link", "https://wiki.example.org/playbook/what-to-do-when-high-latency-strikes"),
					// resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "parent", "some check ID"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.#", "4"),

					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.0.value.#", "1"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.0.value.360613670.absent", "70s"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.0.value.360613670.over.#", "0"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.0.then.#", "1"),
					// Computed:
					// resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.0.then.<computed>.notify.#", "1"),
					// resource.TestCheckResourceAttrSet("circonus_rule_set.icmp-latency-alarm", "if.0.then.<computed>.notify.0"),
					// resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.0.then.<computed>.severity", "1"),

					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.1.value.#", "1"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.1.value.2300199732.over.#", "1"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.1.value.2300199732.over.689776960.last", "120s"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.1.value.2300199732.over.689776960.using", "average"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.1.value.2300199732.min_value", "2"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.1.then.#", "1"),
					// Computed:
					// resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.1.then.<computed>.notify.#", "1"),
					// resource.TestCheckResourceAttrSet("circonus_rule_set.icmp-latency-alarm", "if.1.then.<computed>.notify.0"),
					// resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.1.then.<computed>.severity", "2"),

					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.2.value.#", "1"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.2.value.2842654150.over.#", "1"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.2.value.2842654150.over.999877839.last", "180s"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.2.value.2842654150.over.999877839.using", "average"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.2.value.2842654150.max_value", "300"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.2.then.#", "1"),
					// Computed:
					// resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.2.then.<computed>.notify.#", "1"),
					// resource.TestCheckResourceAttrSet("circonus_rule_set.icmp-latency-alarm", "if.2.then.<computed>.notify.0"),
					// resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.2.then.<computed>.severity", "3"),

					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.3.value.#", "1"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.3.value.803690187.over.#", "0"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.3.value.803690187.max_value", "400"),
					// Computed:
					// resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.3.then.<computed>.notify.#", "1"),
					// resource.TestCheckResourceAttrSet("circonus_rule_set.icmp-latency-alarm", "if.3.then.<computed>.notify.0"),
					// resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.3.then.<computed>.after", "2400s"),
					// resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "if.3.then.<computed>.severity", "4"),

					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "tags.2087084518", "author:terraform"),
					resource.TestCheckResourceAttr("circonus_rule_set.icmp-latency-alarm", "tags.1401442048", "lifecycle:unittest"),
				),
			},
		},
	})
}

func testAccCheckDestroyCirconusRuleSet(s *terraform.State) error {
	ctxt := testAccProvider.Meta().(*providerContext)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "circonus_rule_set" {
			continue
		}

		cid := rs.Primary.ID
		exists, err := checkRuleSetExists(ctxt, api.CIDType(&cid))
		switch {
		case !exists:
			// noop
		case exists:
			return fmt.Errorf("rule set still exists after destroy")
		case err != nil:
			return fmt.Errorf("Error checking rule set: %v", err)
		}
	}

	return nil
}

func checkRuleSetExists(c *providerContext, ruleSetCID api.CIDType) (bool, error) {
	rs, err := c.client.FetchRuleSet(ruleSetCID)
	if err != nil {
		if strings.Contains(err.Error(), defaultCirconus404ErrorString) {
			return false, nil
		}

		return false, err
	}

	if api.CIDType(&rs.CID) == ruleSetCID {
		return true, nil
	}

	return false, nil
}

const testAccCirconusRuleSetConfigFmt = `
variable "test_tags" {
  type = "list"
  default = [ "author:terraform", "lifecycle:unittest" ]
}

resource "circonus_contact_group" "test-trigger" {
  name = "%s"
  tags = [ "${var.test_tags}" ]
}

resource "circonus_check" "api_latency" {
  active = true
  name = "%s"
  period = "60s"

  collector {
    id = "/broker/1"
  }

  icmp_ping {
    count = 1
  }

  metric {
    name = "maximum"
    tags = [ "${var.test_tags}" ]
    type = "numeric"
    unit = "seconds"
  }

  tags = [ "${var.test_tags}" ]
  target = "api.circonus.com"
}

resource "circonus_rule_set" "icmp-latency-alarm" {
  check = "${circonus_check.api_latency.checks[0]}"
  metric_name = "maximum"
  // metric_name = "${circonus_check.api_latency.metric["maximum"].name}"
  // metric_type = "${circonus_check.api_latency.metric["maximum"].type}"
  notes = <<EOF
Simple check to create notifications based on ICMP performance.
EOF
  link = "https://wiki.example.org/playbook/what-to-do-when-high-latency-strikes"
#  parent = "${check cid}"

  if {
    value {
      absent = "70s"
    }

    then {
      notify = [ "${circonus_contact_group.test-trigger.id}" ]
      severity = 1
    }
  }

  if {
    value {
      over {
        last = "120s"
        using = "average"
      }

      min_value = 2
    }

    then {
      notify = [ "${circonus_contact_group.test-trigger.id}" ]
      severity = 2
    }
  }

  if {
    value {
      over {
        last = "180s"
        using = "average"
      }

      max_value = 300
    }

    then {
      notify = [ "${circonus_contact_group.test-trigger.id}" ]
      severity = 3
    }
  }

  if {
    value {
      max_value = 400
    }

    then {
      notify = [ "${circonus_contact_group.test-trigger.id}" ]
      after = "2400s"
      severity = 4
    }
  }

  tags = [ "${var.test_tags}" ]
}
`
