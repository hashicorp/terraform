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

func TestAccAWSWafRegionalIPSet_basic(t *testing.T) {
	var v waf.IPSet
	ipsetName := fmt.Sprintf("ip-set-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalIPSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSWafRegionalIPSetConfig(ipsetName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalIPSetExists("aws_wafregional_ipset.ipset", &v),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "name", ipsetName),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "ip_set_descriptors.4037960608.type", "IPV4"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "ip_set_descriptors.4037960608.value", "192.0.7.0/24"),
				),
			},
		},
	})
}

func TestAccAWSWafRegionalIPSet_disappears(t *testing.T) {
	var v waf.IPSet
	ipsetName := fmt.Sprintf("ip-set-%s", acctest.RandString(5))
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalIPSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRegionalIPSetConfig(ipsetName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalIPSetExists("aws_wafregional_ipset.ipset", &v),
					testAccCheckAWSWafRegionalIPSetDisappears(&v),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAWSWafRegionalIPSet_changeNameForceNew(t *testing.T) {
	var before, after waf.IPSet
	ipsetName := fmt.Sprintf("ip-set-%s", acctest.RandString(5))
	ipsetNewName := fmt.Sprintf("ip-set-new-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalIPSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRegionalIPSetConfig(ipsetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSWafRegionalIPSetExists("aws_wafregional_ipset.ipset", &before),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "name", ipsetName),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "ip_set_descriptors.4037960608.type", "IPV4"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "ip_set_descriptors.4037960608.value", "192.0.7.0/24"),
				),
			},
			{
				Config: testAccAWSWafRegionalIPSetConfigChangeName(ipsetNewName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSWafRegionalIPSetExists("aws_wafregional_ipset.ipset", &after),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "name", ipsetNewName),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "ip_set_descriptors.4037960608.type", "IPV4"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "ip_set_descriptors.4037960608.value", "192.0.7.0/24"),
				),
			},
		},
	})
}

func testAccCheckAWSWafRegionalIPSetDisappears(v *waf.IPSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).wafregionalconn

		wr := newWafRegionalRetryer(conn)
		_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
			req := &waf.UpdateIPSetInput{
				ChangeToken: token,
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

			return conn.UpdateIPSet(req)
		})
		if err != nil {
			return fmt.Errorf("Error Updating WAF IPSet: %s", err)
		}

		_, err = wr.RetryWithToken(func(token *string) (interface{}, error) {
			opts := &waf.DeleteIPSetInput{
				ChangeToken: token,
				IPSetId:     v.IPSetId,
			}
			return conn.DeleteIPSet(opts)
		})
		if err != nil {
			return fmt.Errorf("Error Deleting WAF IPSet: %s", err)
		}
		return nil
	}
}

func testAccCheckAWSWafRegionalIPSetDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_wafregional_ipset" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).wafregionalconn
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

func testAccCheckAWSWafRegionalIPSetExists(n string, v *waf.IPSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No WAF IPSet ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).wafregionalconn
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

func testAccAWSWafRegionalIPSetConfig(name string) string {
	return fmt.Sprintf(`
resource "aws_wafregional_ipset" "ipset" {
  name = "%s"
  ip_set_descriptors {
    type = "IPV4"
    value = "192.0.7.0/24"
  }
}`, name)
}

func testAccAWSWafRegionalIPSetConfigChangeName(name string) string {
	return fmt.Sprintf(`resource "aws_wafregional_ipset" "ipset" {
  name = "%s"
  ip_set_descriptors {
    type = "IPV4"
    value = "192.0.7.0/24"
  }
}`, name)
}

func testAccAWSWafRegionalIPSetConfigChangeIPSetDescriptors(name string) string {
	return fmt.Sprintf(`resource "aws_wafregional_ipset" "ipset" {
  name = "%s"
  ip_set_descriptors {
    type = "IPV4"
    value = "192.0.8.0/24"
  }
}`, name)
}
