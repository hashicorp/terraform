package aws

import (
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIPRanges(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSIPRangesConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccAWSIPRanges("data.aws_ip_ranges.some"),
				),
			},
		},
	})
}

func testAccAWSIPRanges(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		r := s.RootModule().Resources[n]
		a := r.Primary.Attributes

		var (
			cidrBlockSize int
			createDate    time.Time
			err           error
			syncToken     int
		)

		if cidrBlockSize, err = strconv.Atoi(a["cidr_blocks.#"]); err != nil {
			return err
		}

		if cidrBlockSize < 10 {
			return fmt.Errorf("cidr_blocks for eu-west-1 seem suspiciously low: %d", cidrBlockSize)
		}

		if createDate, err = time.Parse("2006-01-02-15-04-05", a["create_date"]); err != nil {
			return err
		}

		if syncToken, err = strconv.Atoi(a["sync_token"]); err != nil {
			return err
		}

		if syncToken != int(createDate.Unix()) {
			return fmt.Errorf("sync_token %d does not match create_date %s", syncToken, createDate)
		}

		var cidrBlocks sort.StringSlice = make([]string, cidrBlockSize)

		for i := range make([]string, cidrBlockSize) {

			block := a[fmt.Sprintf("cidr_blocks.%d", i)]

			if _, _, err := net.ParseCIDR(block); err != nil {
				return fmt.Errorf("malformed CIDR block %s: %s", block, err)
			}

			cidrBlocks[i] = block

		}

		if !sort.IsSorted(cidrBlocks) {
			return fmt.Errorf("unexpected order of cidr_blocks: %s", cidrBlocks)
		}

		var (
			regionMember      = regexp.MustCompile(`regions\.\d+`)
			regions, services int
			serviceMember     = regexp.MustCompile(`services\.\d+`)
		)

		for k, v := range a {

			if regionMember.MatchString(k) {

				if !(v == "eu-west-1" || v == "EU-central-1") {
					return fmt.Errorf("unexpected region %s", v)
				}

				regions = regions + 1

			}

			if serviceMember.MatchString(k) {

				if v != "EC2" {
					return fmt.Errorf("unexpected service %s", v)
				}

				services = services + 1
			}

		}

		if regions != 2 {
			return fmt.Errorf("unexpected number of regions: %d", regions)
		}

		if services != 1 {
			return fmt.Errorf("unexpected number of services: %d", services)
		}

		return nil
	}
}

const testAccAWSIPRangesConfig = `
data "aws_ip_ranges" "some" {
  regions = [ "eu-west-1", "EU-central-1" ]
  services = [ "EC2" ]
}
`
