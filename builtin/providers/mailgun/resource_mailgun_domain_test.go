package mailgun

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pearkes/mailgun"
)

func TestAccMailgunDomain_Basic(t *testing.T) {
	var domain mailgun.Domain

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMailgunDomainDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckMailgunDomainConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMailgunDomainExists("mailgun_domain.foobar", &domain),
					testAccCheckMailgunDomainAttributes(&domain),
					resource.TestCheckResourceAttr(
						"mailgun_domain.foobar", "name", "terraform.example.com"),
					resource.TestCheckResourceAttr(
						"mailgun_domain.foobar", "spam_action", "disabled"),
					resource.TestCheckResourceAttr(
						"mailgun_domain.foobar", "smtp_password", "foobar"),
					resource.TestCheckResourceAttr(
						"mailgun_domain.foobar", "wildcard", "true"),
				),
			},
		},
	})
}

func testAccCheckMailgunDomainDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*mailgun.Client)

	for _, rs := range s.Resources {
		if rs.Type != "mailgun_domain" {
			continue
		}

		_, err := client.RetrieveDomain(rs.ID)

		if err == nil {
			return fmt.Errorf("Domain still exists")
		}
	}

	return nil
}

func testAccCheckMailgunDomainAttributes(Domain *mailgun.Domain) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if Domain.Name != "terraform.example.com" {
			return fmt.Errorf("Bad name: %s", Domain.Name)
		}

		if Domain.SpamAction != "disabled" {
			return fmt.Errorf("Bad spam_action: %s", Domain.SpamAction)
		}

		if Domain.Wildcard != true {
			return fmt.Errorf("Bad wildcard: %s", Domain.Wildcard)
		}

		if Domain.SmtpPassword != "foobar" {
			return fmt.Errorf("Bad smtp_password: %s", Domain.SmtpPassword)
		}

		return nil
	}
}

func testAccCheckMailgunDomainExists(n string, Domain *mailgun.Domain) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No Domain ID is set")
		}

		client := testAccProvider.Meta().(*mailgun.Client)

		foundDomain, err := client.RetrieveDomain(rs.ID)

		if err != nil {
			return err
		}

		if foundDomain.Name != rs.ID {
			return fmt.Errorf("Domain not found")
		}

		*Domain = foundDomain

		return nil
	}
}

const testAccCheckMailgunDomainConfig_basic = `
resource "mailgun_domain" "foobar" {
    name = "terraform.example.com"
    spam_action = "disabled"
    smtp_password = "foobar"
    wildcard = true
}`
