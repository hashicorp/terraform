package mailgun

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pearkes/mailgun"
)

func TestAccMailgunDomain_Basic(t *testing.T) {
	var resp mailgun.DomainResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMailgunDomainDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckMailgunDomainConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMailgunDomainExists("mailgun_domain.foobar", &resp),
					testAccCheckMailgunDomainAttributes(&resp),
					resource.TestCheckResourceAttr(
						"mailgun_domain.foobar", "name", "terraform.example.com"),
					resource.TestCheckResourceAttr(
						"mailgun_domain.foobar", "spam_action", "disabled"),
					resource.TestCheckResourceAttr(
						"mailgun_domain.foobar", "smtp_password", "foobar"),
					resource.TestCheckResourceAttr(
						"mailgun_domain.foobar", "wildcard", "true"),
					resource.TestCheckResourceAttr(
						"mailgun_domain.foobar", "receiving_records.0.priority", "10"),
					resource.TestCheckResourceAttr(
						"mailgun_domain.foobar", "sending_records.0.name", "terraform.example.com"),
				),
			},
		},
	})
}

func testAccCheckMailgunDomainDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*mailgun.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "mailgun_domain" {
			continue
		}

		_, err := client.RetrieveDomain(rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Domain still exists")
		}
	}

	return nil
}

func testAccCheckMailgunDomainAttributes(DomainResp *mailgun.DomainResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if DomainResp.Domain.Name != "terraform.example.com" {
			return fmt.Errorf("Bad name: %s", DomainResp.Domain.Name)
		}

		if DomainResp.Domain.SpamAction != "disabled" {
			return fmt.Errorf("Bad spam_action: %s", DomainResp.Domain.SpamAction)
		}

		if DomainResp.Domain.Wildcard != true {
			return fmt.Errorf("Bad wildcard: %t", DomainResp.Domain.Wildcard)
		}

		if DomainResp.Domain.SmtpPassword != "foobar" {
			return fmt.Errorf("Bad smtp_password: %s", DomainResp.Domain.SmtpPassword)
		}

		if DomainResp.ReceivingRecords[0].Priority == "" {
			return fmt.Errorf("Bad receiving_records: %s", DomainResp.ReceivingRecords)
		}

		if DomainResp.SendingRecords[0].Name == "" {
			return fmt.Errorf("Bad sending_records: %s", DomainResp.SendingRecords)
		}

		return nil
	}
}

func testAccCheckMailgunDomainExists(n string, DomainResp *mailgun.DomainResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Domain ID is set")
		}

		client := testAccProvider.Meta().(*mailgun.Client)

		resp, err := client.RetrieveDomain(rs.Primary.ID)

		if err != nil {
			return err
		}

		if resp.Domain.Name != rs.Primary.ID {
			return fmt.Errorf("Domain not found")
		}

		*DomainResp = resp

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
