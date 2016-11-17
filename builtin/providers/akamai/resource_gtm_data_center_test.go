package akamai

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAkamaiGtmDatacenterBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAkamaiGTMDataCenterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckAkamaiGTMDataCenterConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAkamaiGTMDataCenterExists("akamai_gtm_data_center.test_dc"),
					resource.TestCheckResourceAttr("akamai_gtm_data_center.test_dc", "name", "test_dc"),
					resource.TestCheckResourceAttr("akamai_gtm_data_center.test_dc", "domain", "terraform-test.akadns.net"),
					resource.TestCheckResourceAttr("akamai_gtm_data_center.test_dc", "city", "Downpatrick"),
					resource.TestCheckResourceAttr("akamai_gtm_data_center.test_dc", "country", "GB"),
					resource.TestCheckResourceAttr("akamai_gtm_data_center.test_dc", "continent", "EU"),
				),
			},
		},
	})
}

func testAccAkamaiGTMDataCenterDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Clients).GTM

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "akamai_gtm_data_center" {
			continue
		}
		dcId, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		// Try to find the data center
		_, err = client.DataCenter("terraform-test.akadns.net", dcId)

		if err == nil {
			fmt.Errorf("Data center still exists")
		}
	}

	return nil
}

func testAccCheckAkamaiGTMDataCenterExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("n is %s", n)
			return fmt.Errorf("Not found %s", rs)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*Clients).GTM
		dcId, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		readDc, err := client.DataCenter("terraform-test.akadns.net", dcId)

		if err != nil {
			return err
		}

		if strconv.Itoa(readDc.DataCenterID) != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		return nil
	}
}

const testAccCheckAkamaiGTMDataCenterConfigBasic = `
resource "akamai_gtm_data_center" "test_dc" {
  name =  "test_dc"
	domain = "terraform-test.akadns.net"
	city = "Downpatrick"
	country = "GB"
	continent = "EU"
	latitude = 54.367
	longitude = -5.582
	virtual = false
}`
