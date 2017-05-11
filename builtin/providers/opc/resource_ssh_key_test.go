package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOPCSSHKey_basic(t *testing.T) {
	ruleResourceName := "opc_compute_ssh_key.test"
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOPCSSHKeyBasic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckSSHKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccOPCCheckSSHKeyExists,
					resource.TestCheckResourceAttr(ruleResourceName, "key", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7Wa2OClh4LDCpR4A1x251PfzeUHvA3uo3Z4joYKIlQXP6242588bq6eh79ihm+HZAuxNoIkkS4OMIelUtiHcYSMYK7niXpato3cUdQHXjwchZjc3wwcXC/hAWK2QJkO7yLgCuYMTqyz2saZ/9zW12QS24rJH1DKFDbq4V40+HF7PQoq6G40Dp0X+slZri223pHJiqHKlyhUZuvMar7QnLZlZ7jenPyqVSpY7IC5KPj6geQSD2tSnVKjRo4TWVkIexSo6iHEu5vzcjVYGBw9RVGhmOd8pCcbB85M01MJFdbqLMjUHREE7/t767hmem3YdSPhMvnbBNPb7VSB+8ZQKn"),
				),
			},
		},
	})
}

func TestAccOPCSSHKey_update(t *testing.T) {
	ruleResourceName := "opc_compute_ssh_key.test"
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOPCSSHKeyBasic, ri)
	updatedConfig := fmt.Sprintf(testAccOPCSSHKeyUpdated, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckSSHKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccOPCCheckSSHKeyExists,
					resource.TestCheckResourceAttr(ruleResourceName, "key", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7Wa2OClh4LDCpR4A1x251PfzeUHvA3uo3Z4joYKIlQXP6242588bq6eh79ihm+HZAuxNoIkkS4OMIelUtiHcYSMYK7niXpato3cUdQHXjwchZjc3wwcXC/hAWK2QJkO7yLgCuYMTqyz2saZ/9zW12QS24rJH1DKFDbq4V40+HF7PQoq6G40Dp0X+slZri223pHJiqHKlyhUZuvMar7QnLZlZ7jenPyqVSpY7IC5KPj6geQSD2tSnVKjRo4TWVkIexSo6iHEu5vzcjVYGBw9RVGhmOd8pCcbB85M01MJFdbqLMjUHREE7/t767hmem3YdSPhMvnbBNPb7VSB+8ZQKn"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccOPCCheckSSHKeyExists,
					resource.TestCheckResourceAttr(ruleResourceName, "key", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDeXD/4cetIxK3a/mNbE8F0oYFOicK07Am6YyS0tV4Etak29fB2FoRwGAMETN0w7kKa8nKyvjZBH2mTkdAELoSbB70yZLNSufK7GMyLQXRG8c51xFDhTjLXZ92zSN6ZZBrnc7Z7iXCHsfAyXcrTmv9jgm3nE0QF1/AJgHXNa6GqzsyjilKkRjQBhUTqkTQylyVytPJHgM5W/v2vStfFK5wY9h9oDiHJiNACPOxE8v9A+u9MnKaq+E6AuarA0VQJbPWqVzoHWMUXL0ck+WYfZyX17VPB6c18h4Wn27lNxCEE7jaMLIVbMpAW5ICW1UVnrT6/ZoSTseJjEBlukPlZVQu7"),
				),
			},
		},
	})
}

func TestAccOPCSSHKey_disable(t *testing.T) {
	ruleResourceName := "opc_compute_ssh_key.test"
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOPCSSHKeyBasic, ri)
	updatedConfig := fmt.Sprintf(testAccOPCSSHKeyDisabled, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckSSHKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccOPCCheckSSHKeyExists,
					resource.TestCheckResourceAttr(ruleResourceName, "enabled", "true"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccOPCCheckSSHKeyExists,
					resource.TestCheckResourceAttr(ruleResourceName, "enabled", "false"),
				),
			},
		},
	})
}

func testAccOPCCheckSSHKeyExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).SSHKeys()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_ssh_key" {
			continue
		}

		input := compute.GetSSHKeyInput{
			Name: rs.Primary.Attributes["name"],
		}
		if _, err := client.GetSSHKey(&input); err != nil {
			return fmt.Errorf("Error retrieving state of SSH Key %s: %s", input.Name, err)
		}
	}

	return nil
}

func testAccOPCCheckSSHKeyDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).SSHKeys()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_ssh_key" {
			continue
		}

		input := compute.GetSSHKeyInput{
			Name: rs.Primary.Attributes["name"],
		}
		if info, err := client.GetSSHKey(&input); err == nil {
			return fmt.Errorf("SSH Key %s still exists: %#v", input.Name, info)
		}
	}

	return nil
}

const testAccOPCSSHKeyBasic = `
resource "opc_compute_ssh_key" "test" {
	name    = "acc-ssh-key-%d"
	key     = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7Wa2OClh4LDCpR4A1x251PfzeUHvA3uo3Z4joYKIlQXP6242588bq6eh79ihm+HZAuxNoIkkS4OMIelUtiHcYSMYK7niXpato3cUdQHXjwchZjc3wwcXC/hAWK2QJkO7yLgCuYMTqyz2saZ/9zW12QS24rJH1DKFDbq4V40+HF7PQoq6G40Dp0X+slZri223pHJiqHKlyhUZuvMar7QnLZlZ7jenPyqVSpY7IC5KPj6geQSD2tSnVKjRo4TWVkIexSo6iHEu5vzcjVYGBw9RVGhmOd8pCcbB85M01MJFdbqLMjUHREE7/t767hmem3YdSPhMvnbBNPb7VSB+8ZQKn"
	enabled = true
}
`

const testAccOPCSSHKeyUpdated = `
resource "opc_compute_ssh_key" "test" {
	name    = "acc-ssh-key-%d"
	key     = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDeXD/4cetIxK3a/mNbE8F0oYFOicK07Am6YyS0tV4Etak29fB2FoRwGAMETN0w7kKa8nKyvjZBH2mTkdAELoSbB70yZLNSufK7GMyLQXRG8c51xFDhTjLXZ92zSN6ZZBrnc7Z7iXCHsfAyXcrTmv9jgm3nE0QF1/AJgHXNa6GqzsyjilKkRjQBhUTqkTQylyVytPJHgM5W/v2vStfFK5wY9h9oDiHJiNACPOxE8v9A+u9MnKaq+E6AuarA0VQJbPWqVzoHWMUXL0ck+WYfZyX17VPB6c18h4Wn27lNxCEE7jaMLIVbMpAW5ICW1UVnrT6/ZoSTseJjEBlukPlZVQu7"
	enabled = true
}
`

const testAccOPCSSHKeyDisabled = `
resource "opc_compute_ssh_key" "test" {
	name    = "acc-ssh-key-%d"
	key     = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7Wa2OClh4LDCpR4A1x251PfzeUHvA3uo3Z4joYKIlQXP6242588bq6eh79ihm+HZAuxNoIkkS4OMIelUtiHcYSMYK7niXpato3cUdQHXjwchZjc3wwcXC/hAWK2QJkO7yLgCuYMTqyz2saZ/9zW12QS24rJH1DKFDbq4V40+HF7PQoq6G40Dp0X+slZri223pHJiqHKlyhUZuvMar7QnLZlZ7jenPyqVSpY7IC5KPj6geQSD2tSnVKjRo4TWVkIexSo6iHEu5vzcjVYGBw9RVGhmOd8pCcbB85M01MJFdbqLMjUHREE7/t767hmem3YdSPhMvnbBNPb7VSB+8ZQKn"
	enabled = false
}
`
