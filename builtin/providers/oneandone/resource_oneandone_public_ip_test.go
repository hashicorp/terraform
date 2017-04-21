package oneandone

import (
	"fmt"
	"testing"

	"github.com/1and1/oneandone-cloudserver-sdk-go"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"os"
	"time"
)

func TestAccOneandonePublicIp_Basic(t *testing.T) {
	var public_ip oneandone.PublicIp

	reverse_dns := "example.de"
	reverse_dns_updated := "example.ba"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDOneandonePublicIpDestroyCheck,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckOneandonePublicIp_basic, reverse_dns),
				Check: resource.ComposeTestCheckFunc(
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					testAccCheckOneandonePublicIpExists("oneandone_public_ip.ip", &public_ip),
					testAccCheckOneandonePublicIpAttributes("oneandone_public_ip.ip", reverse_dns),
					resource.TestCheckResourceAttr("oneandone_public_ip.ip", "reverse_dns", reverse_dns),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckOneandonePublicIp_basic, reverse_dns_updated),
				Check: resource.ComposeTestCheckFunc(
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					testAccCheckOneandonePublicIpExists("oneandone_public_ip.ip", &public_ip),
					testAccCheckOneandonePublicIpAttributes("oneandone_public_ip.ip", reverse_dns_updated),
					resource.TestCheckResourceAttr("oneandone_public_ip.ip", "reverse_dns", reverse_dns_updated),
				),
			},
		},
	})
}

func testAccCheckDOneandonePublicIpDestroyCheck(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "oneandone_public_ip" {
			continue
		}

		api := oneandone.New(os.Getenv("ONEANDONE_TOKEN"), oneandone.BaseUrl)

		_, err := api.GetPublicIp(rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Public IP still exists %s %s", rs.Primary.ID, err.Error())
		}
	}

	return nil
}
func testAccCheckOneandonePublicIpAttributes(n string, reverse_dns string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.Attributes["reverse_dns"] != reverse_dns {
			return fmt.Errorf("Bad name: expected %s : found %s ", reverse_dns, rs.Primary.Attributes["name"])
		}

		return nil
	}
}

func testAccCheckOneandonePublicIpExists(n string, public_ip *oneandone.PublicIp) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		api := oneandone.New(os.Getenv("ONEANDONE_TOKEN"), oneandone.BaseUrl)

		found_public_ip, err := api.GetPublicIp(rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("Error occured while fetching public IP: %s", rs.Primary.ID)
		}
		if found_public_ip.Id != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}
		public_ip = found_public_ip

		return nil
	}
}

const testAccCheckOneandonePublicIp_basic = `
resource "oneandone_public_ip" "ip" {
	"ip_type" = "IPV4"
	"reverse_dns" = "%s"
	"datacenter" = "GB"
}`
