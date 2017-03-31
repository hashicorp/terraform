package circonus

import (
	"fmt"
	"strings"
	"testing"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccCirconusContactGroup_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusContactGroup,
		Steps: []resource.TestStep{
			{
				Config: testAccCirconusContactGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					// testAccContactGroupExists("circonus_contact_group.staging-sev3", "foo"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "name", "ops-staging-sev3"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "email.#", "3"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "email.1119127802.address", ""),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "email.1119127802.user", "/user/5469"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "email.1456570992.address", ""),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "email.1456570992.user", "/user/6331"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "email.343263208.address", "user@example.com"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "email.343263208.user", ""),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "http.#", "1"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "http.1287846151.address", "https://www.example.org/post/endpoint"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "http.1287846151.format", "json"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "http.1287846151.method", "POST"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "irc.#", "0"),
					// resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "irc.918937268.user", "/user/6331"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "slack.#", "1"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "slack.274933206.channel", "#ops-staging"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "slack.274933206.team", "T123UT98F"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "slack.274933206.username", "Circonus"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "slack.274933206.buttons", "true"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "sms.#", "1"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "sms.1119127802.user", "/user/5469"),

					// xmpp.# will be 0 for user faux user accounts that don't have an
					// XMPP address setup.
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "xmpp.#", "0"),
					// resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "xmpp.1119127802.user", "/user/5469"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "victorops.#", "1"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "victorops.2029434450.api_key", "123"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "victorops.2029434450.critical", "2"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "victorops.2029434450.info", "5"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "victorops.2029434450.team", "bender"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "victorops.2029434450.warning", "3"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "aggregation_window", "60s"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.#", "5"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.689365425.severity", "1"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.689365425.reminder", "60s"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.689365425.escalate_after", "3600s"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.689365425.escalate_to", "/contact_group/2913"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.551050940.severity", "2"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.551050940.reminder", "120s"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.551050940.escalate_after", "7200s"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.551050940.escalate_to", "/contact_group/2913"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.1292974544.severity", "3"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.1292974544.reminder", "180s"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.1292974544.escalate_after", "10800s"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.1292974544.escalate_to", "/contact_group/2913"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.1183354841.severity", "4"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.1183354841.reminder", "240s"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.1183354841.escalate_after", "14400s"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.1183354841.escalate_to", "/contact_group/2913"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.2942620849.severity", "5"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.2942620849.reminder", "300s"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.2942620849.escalate_after", "18000s"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "alert_option.2942620849.escalate_to", "/contact_group/2913"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "long_message", "a long message"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "long_subject", "long subject"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "long_summary", "long summary"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "short_message", "short message"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "short_summary", "short summary"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "tags.2087084518", "author:terraform"),
					resource.TestCheckResourceAttr("circonus_contact_group.staging-sev3", "tags.393923453", "other:foo"),
				),
			},
		},
	})
}

func testAccCheckDestroyCirconusContactGroup(s *terraform.State) error {
	c := testAccProvider.Meta().(*providerContext)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "circonus_contact_group" {
			continue
		}

		cid := rs.Primary.ID
		exists, err := checkContactGroupExists(c, api.CIDType(&cid))
		switch {
		case !exists:
			// noop
		case exists:
			return fmt.Errorf("contact group still exists after destroy")
		case err != nil:
			return fmt.Errorf("Error checking contact group %s", err)
		}
	}

	return nil
}

func checkContactGroupExists(c *providerContext, contactGroupCID api.CIDType) (bool, error) {
	cb, err := c.client.FetchContactGroup(contactGroupCID)
	if err != nil {
		if strings.Contains(err.Error(), defaultCirconus404ErrorString) {
			return false, nil
		}

		return false, err
	}

	if api.CIDType(&cb.CID) == contactGroupCID {
		return true, nil
	}

	return false, nil
}

const testAccCirconusContactGroupConfig = `
resource "circonus_contact_group" "staging-sev3" {
  name = "ops-staging-sev3"

  email {
    user = "/user/5469"
  }

  email {
    address = "user@example.com"
  }

  email {
    user = "/user/6331"
  }

  http {
    address = "https://www.example.org/post/endpoint"
    format = "json"
    method = "POST"
  }

/*
  // Account needs to be setup with IRC before this can work.
  irc {
    user = "/user/6331"
  }
*/

/*
  pager_duty {
    // NOTE(sean@): needs to be filled in
  }
*/

  slack {
    channel = "#ops-staging"
    team = "T123UT98F"
    username = "Circonus"
    buttons = true
  }

  sms {
    user = "/user/5469"
  }

  victorops {
    api_key = "123"
    critical = 2
    info = 5
    team = "bender"
    warning = 3
  }

	// Faux user accounts that don't have an XMPP address setup will not return a
	// valid response in the future.
  //
  // xmpp {
  //   user = "/user/5469"
  // }

  aggregation_window = "1m"

  alert_option {
    severity = 1
    reminder = "60s"
    escalate_after = "3600s"
    escalate_to = "/contact_group/2913"
  }

  alert_option {
    severity = 2
    reminder = "2m"
    escalate_after = "2h"
    escalate_to = "/contact_group/2913"
  }

  alert_option {
    severity = 3
    reminder = "3m"
    escalate_after = "3h"
    escalate_to = "/contact_group/2913"
  }

  alert_option {
    severity = 4
    reminder = "4m"
    escalate_after = "4h"
    escalate_to = "/contact_group/2913"
  }

  alert_option {
    severity = 5
    reminder = "5m"
    escalate_after = "5h"
    escalate_to = "/contact_group/2913"
  }

  // alert_formats: omit to use defaults
  long_message = "a long message"
  long_subject = "long subject"
  long_summary = "long summary"
  short_message = "short message"
  short_summary = "short summary"

  tags = [
    "author:terraform",
    "other:foo",
  ]
}
`
