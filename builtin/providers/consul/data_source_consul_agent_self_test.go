package consul

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataConsulAgentSelf_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataConsulAgentSelfConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataSourceValue("data.consul_agent_self.read", "acl_datacenter", "<all>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "acl_default_policy", "<all>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "acl_disabled_ttl", "<all>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "acl_down_policy", "<all>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "acl_enforce_0_8_semantics", "<all>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "acl_ttl", "<all>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "advertise_addr", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "bind_addr", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "bootstrap_expect", "<all>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "bootstrap_mode", "false"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "client_addr", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "datacenter", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "dev_mode", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "domain", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "enable_anonymous_signature", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "enable_coordinates", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "enable_debug", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "enable_remote_exec", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "enable_syslog", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "enable_ui", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "enable_update_check", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "id", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "leave_on_int", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "leave_on_term", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "log_level", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "name", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "pid_file", "<all>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "rejoin_after_leave", "<any>"),
					// testAccCheckDataSourceValue("data.consul_agent_self.read", "retry_join", "<all>"),
					// testAccCheckDataSourceValue("data.consul_agent_self.read", "retry_join_wan", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "retry_max_attempts", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "retry_max_attempts_wan", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "serf_lan_bind_addr", "<all>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "serf_wan_bind_addr", "<all>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "server_mode", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "server_name", "<all>"),
					// testAccCheckDataSourceValue("data.consul_agent_self.read", "start_join", "<all>"),
					// testAccCheckDataSourceValue("data.consul_agent_self.read", "start_join_wan", "<all>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "syslog_facility", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "telemetry.enable_hostname", "<all>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "tls_ca_file", "<all>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "tls_cert_file", "<all>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "tls_key_file", "<all>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "tls_verify_incoming", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "tls_verify_outgoing", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "tls_verify_server_hostname", "<any>"),
				),
			},
		},
	})
}

func testAccCheckDataSourceValue(n, attr, val string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rn, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Resource not found")
		}
		out, found := rn.Primary.Attributes[attr]
		switch {
		case !found:
			return fmt.Errorf("Attribute '%s' not found: %#v", attr, rn.Primary.Attributes)
		case val == "<all>":
			// Value found, don't care what the payload is (including the zero value)
		case val != "<any>" && out != val:
			return fmt.Errorf("Attribute '%s' value '%s' != '%s'", attr, out, val)
		case val == "<any>" && out == "":
			return fmt.Errorf("Attribute '%s' value '%s'", attr, out)
		}
		return nil
	}
}

const testAccDataConsulAgentSelfConfig = `
data "consul_agent_self" "read" {
}
`
