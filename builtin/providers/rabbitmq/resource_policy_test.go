package rabbitmq

import (
	"fmt"
	"strings"
	"testing"

	"github.com/michaelklishin/rabbit-hole"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPolicy(t *testing.T) {
	var policy rabbithole.Policy
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPolicyCheckDestroy(&policy),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccPolicyConfig_basic,
				Check: testAccPolicyCheck(
					"rabbitmq_policy.test", &policy,
				),
			},
			resource.TestStep{
				Config: testAccPolicyConfig_update,
				Check: testAccPolicyCheck(
					"rabbitmq_policy.test", &policy,
				),
			},
		},
	})
}

func testAccPolicyCheck(rn string, policy *rabbithole.Policy) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("policy id not set")
		}

		rmqc := testAccProvider.Meta().(*rabbithole.Client)
		policyParts := strings.Split(rs.Primary.ID, "@")

		policies, err := rmqc.ListPolicies()
		if err != nil {
			return fmt.Errorf("Error retrieving policies: %s", err)
		}

		for _, p := range policies {
			if p.Name == policyParts[0] && p.Vhost == policyParts[1] {
				policy = &p
				return nil
			}
		}

		return fmt.Errorf("Unable to find policy %s", rn)
	}
}

func testAccPolicyCheckDestroy(policy *rabbithole.Policy) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rmqc := testAccProvider.Meta().(*rabbithole.Client)

		policies, err := rmqc.ListPolicies()
		if err != nil {
			return fmt.Errorf("Error retrieving policies: %s", err)
		}

		for _, p := range policies {
			if p.Name == policy.Name && p.Vhost == policy.Vhost {
				return fmt.Errorf("Policy %s@%s still exist", policy.Name, policy.Vhost)
			}
		}

		return nil
	}
}

const testAccPolicyConfig_basic = `
resource "rabbitmq_vhost" "test" {
    name = "test"
}

resource "rabbitmq_permissions" "guest" {
    user = "guest"
    vhost = "${rabbitmq_vhost.test.name}"
    permissions {
        configure = ".*"
        write = ".*"
        read = ".*"
    }
}

resource "rabbitmq_policy" "test" {
    name = "test"
    vhost = "${rabbitmq_permissions.guest.vhost}"
    policy {
        pattern = ".*"
        priority = 0
        apply_to = "all"
        definition {
            ha-mode = "nodes"
            ha-params = "a,b,c"
        }
    }
}`

const testAccPolicyConfig_update = `
resource "rabbitmq_vhost" "test" {
    name = "test"
}

resource "rabbitmq_permissions" "guest" {
    user = "guest"
    vhost = "${rabbitmq_vhost.test.name}"
    permissions {
        configure = ".*"
        write = ".*"
        read = ".*"
    }
}

resource "rabbitmq_policy" "test" {
    name = "test"
    vhost = "${rabbitmq_permissions.guest.vhost}"
    policy {
        pattern = ".*"
        priority = 0
        apply_to = "all"
        definition {
            ha-mode = "all"
        }
    }
}`
