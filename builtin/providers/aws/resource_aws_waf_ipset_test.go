package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/terraform/helper/acctest"
)

func TestAccAWSWafIPSet_basic(t *testing.T) {
	var v waf.IPSet
	ipsetName := fmt.Sprintf("ip-set-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafIPSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSWafIPSetConfig(ipsetName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafIPSetExists("aws_waf_ipset.ipset", &v),
					resource.TestCheckResourceAttr(
						"aws_waf_ipset.ipset", "name", ipsetName),
					resource.TestCheckResourceAttr(
						"aws_waf_ipset.ipset", "ip_set_descriptors.4037960608.type", "IPV4"),
					resource.TestCheckResourceAttr(
						"aws_waf_ipset.ipset", "ip_set_descriptors.4037960608.value", "192.0.7.0/24"),
				),
			},
		},
	})
}

func TestAccAWSWafIPSet_disappears(t *testing.T) {
	var v waf.IPSet
	ipsetName := fmt.Sprintf("ip-set-%s", acctest.RandString(5))
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafIPSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafIPSetConfig(ipsetName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafIPSetExists("aws_waf_ipset.ipset", &v),
					testAccCheckAWSWafIPSetDisappears(&v),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAWSWafIPSet_changeNameForceNew(t *testing.T) {
	var before, after waf.IPSet
	ipsetName := fmt.Sprintf("ip-set-%s", acctest.RandString(5))
	ipsetNewName := fmt.Sprintf("ip-set-new-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafIPSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafIPSetConfig(ipsetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSWafIPSetExists("aws_waf_ipset.ipset", &before),
					resource.TestCheckResourceAttr(
						"aws_waf_ipset.ipset", "name", ipsetName),
					resource.TestCheckResourceAttr(
						"aws_waf_ipset.ipset", "ip_set_descriptors.4037960608.type", "IPV4"),
					resource.TestCheckResourceAttr(
						"aws_waf_ipset.ipset", "ip_set_descriptors.4037960608.value", "192.0.7.0/24"),
				),
			},
			{
				Config: testAccAWSWafIPSetConfigChangeName(ipsetNewName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSWafIPSetExists("aws_waf_ipset.ipset", &after),
					resource.TestCheckResourceAttr(
						"aws_waf_ipset.ipset", "name", ipsetNewName),
					resource.TestCheckResourceAttr(
						"aws_waf_ipset.ipset", "ip_set_descriptors.4037960608.type", "IPV4"),
					resource.TestCheckResourceAttr(
						"aws_waf_ipset.ipset", "ip_set_descriptors.4037960608.value", "192.0.7.0/24"),
				),
			},
		},
	})
}

func testAccCheckAWSWafIPSetDisappears(v *waf.IPSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).wafconn

		// ChangeToken
		var ct *waf.GetChangeTokenInput

		resp, err := conn.GetChangeToken(ct)
		if err != nil {
			return fmt.Errorf("Error getting change token: %s", err)
		}

		req := &waf.UpdateIPSetInput{
			ChangeToken: resp.ChangeToken,
			IPSetId:     v.IPSetId,
		}

		for _, IPSetDescriptor := range v.IPSetDescriptors {
			IPSetUpdate := &waf.IPSetUpdate{
				Action: aws.String("DELETE"),
				IPSetDescriptor: &waf.IPSetDescriptor{
					Type:  IPSetDescriptor.Type,
					Value: IPSetDescriptor.Value,
				},
			}
			req.Updates = append(req.Updates, IPSetUpdate)
		}

		_, err = conn.UpdateIPSet(req)
		if err != nil {
			return fmt.Errorf("Error Updating WAF IPSet: %s", err)
		}

		resp, err = conn.GetChangeToken(ct)
		if err != nil {
			return fmt.Errorf("Error getting change token for waf IPSet: %s", err)
		}

		opts := &waf.DeleteIPSetInput{
			ChangeToken: resp.ChangeToken,
			IPSetId:     v.IPSetId,
		}
		if _, err := conn.DeleteIPSet(opts); err != nil {
			return err
		}
		return nil
	}
}

func testAccCheckAWSWafIPSetDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_waf_ipset" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).wafconn
		resp, err := conn.GetIPSet(
			&waf.GetIPSetInput{
				IPSetId: aws.String(rs.Primary.ID),
			})

		if err == nil {
			if *resp.IPSet.IPSetId == rs.Primary.ID {
				return fmt.Errorf("WAF IPSet %s still exists", rs.Primary.ID)
			}
		}

		// Return nil if the IPSet is already destroyed
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "WAFNonexistentItemException" {
				return nil
			}
		}

		return err
	}

	return nil
}

func testAccCheckAWSWafIPSetExists(n string, v *waf.IPSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No WAF IPSet ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).wafconn
		resp, err := conn.GetIPSet(&waf.GetIPSetInput{
			IPSetId: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		if *resp.IPSet.IPSetId == rs.Primary.ID {
			*v = *resp.IPSet
			return nil
		}

		return fmt.Errorf("WAF IPSet (%s) not found", rs.Primary.ID)
	}
}

func testAccAWSWafIPSetConfig(name string) string {
	return fmt.Sprintf(`
resource "aws_waf_ipset" "ipset" {
  name = "%s"
  ip_set_descriptors {
    type = "IPV4"
    value = "192.0.7.0/24"
  }
}`, name)
}

func testAccAWSWafIPSetConfigChangeName(name string) string {
	return fmt.Sprintf(`resource "aws_waf_ipset" "ipset" {
  name = "%s"
  ip_set_descriptors {
    type = "IPV4"
    value = "192.0.7.0/24"
  }
}`, name)
}

func testAccAWSWafIPSetConfigChangeIPSetDescriptors(name string) string {
	return fmt.Sprintf(`resource "aws_waf_ipset" "ipset" {
  name = "%s"
  ip_set_descriptors {
    type = "IPV4"
    value = "192.0.8.0/24"
  }
}`, name)
}
