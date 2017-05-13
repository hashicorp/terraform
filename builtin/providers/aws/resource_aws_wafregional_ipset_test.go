package aws

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/aws/aws-sdk-go/service/wafregional"
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
						"aws_wafregional_ipset.ipset", "ip_set_descriptor.4037960608.type", "IPV4"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "ip_set_descriptor.4037960608.value", "192.0.7.0/24"),
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
						"aws_wafregional_ipset.ipset", "ip_set_descriptor.4037960608.type", "IPV4"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "ip_set_descriptor.4037960608.value", "192.0.7.0/24"),
				),
			},
			{
				Config: testAccAWSWafRegionalIPSetConfigChangeName(ipsetNewName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSWafRegionalIPSetExists("aws_wafregional_ipset.ipset", &after),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "name", ipsetNewName),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "ip_set_descriptor.4037960608.type", "IPV4"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "ip_set_descriptor.4037960608.value", "192.0.7.0/24"),
				),
			},
		},
	})
}

func TestAccAWSWafRegionalIPSet_changeDescriptors(t *testing.T) {
	var before, after waf.IPSet
	ipsetName := fmt.Sprintf("ip-set-%s", acctest.RandString(5))

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
						"aws_wafregional_ipset.ipset", "ip_set_descriptor.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "ip_set_descriptor.4037960608.type", "IPV4"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "ip_set_descriptor.4037960608.value", "192.0.7.0/24"),
				),
			},
			{
				Config: testAccAWSWafRegionalIPSetConfigChangeIPSetDescriptors(ipsetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSWafRegionalIPSetExists("aws_wafregional_ipset.ipset", &after),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "name", ipsetName),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "ip_set_descriptor.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "ip_set_descriptor.115741513.type", "IPV4"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "ip_set_descriptor.115741513.value", "192.0.8.0/24"),
				),
			},
		},
	})
}

func TestAccAWSWafRegionalIPSet_noDescriptors(t *testing.T) {
	var ipset waf.IPSet
	ipsetName := fmt.Sprintf("ip-set-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalIPSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRegionalIPSetConfig_noDescriptors(ipsetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSWafRegionalIPSetExists("aws_wafregional_ipset.ipset", &ipset),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "name", ipsetName),
					resource.TestCheckResourceAttr(
						"aws_wafregional_ipset.ipset", "ip_set_descriptor.#", "0"),
				),
			},
		},
	})
}

func TestDiffWafRegionalIpSetDescriptors(t *testing.T) {
	testCases := []struct {
		Old             []interface{}
		New             []interface{}
		ExpectedUpdates []*waf.IPSetUpdate
	}{
		{
			// Change
			Old: []interface{}{
				map[string]interface{}{"type": "IPV4", "value": "192.0.7.0/24"},
			},
			New: []interface{}{
				map[string]interface{}{"type": "IPV4", "value": "192.0.8.0/24"},
			},
			ExpectedUpdates: []*waf.IPSetUpdate{
				&waf.IPSetUpdate{
					Action: aws.String(wafregional.ChangeActionDelete),
					IPSetDescriptor: &waf.IPSetDescriptor{
						Type:  aws.String("IPV4"),
						Value: aws.String("192.0.7.0/24"),
					},
				},
				&waf.IPSetUpdate{
					Action: aws.String(wafregional.ChangeActionInsert),
					IPSetDescriptor: &waf.IPSetDescriptor{
						Type:  aws.String("IPV4"),
						Value: aws.String("192.0.8.0/24"),
					},
				},
			},
		},
		{
			// Fresh IPSet
			Old: []interface{}{},
			New: []interface{}{
				map[string]interface{}{"type": "IPV4", "value": "10.0.1.0/24"},
				map[string]interface{}{"type": "IPV4", "value": "10.0.2.0/24"},
				map[string]interface{}{"type": "IPV4", "value": "10.0.3.0/24"},
			},
			ExpectedUpdates: []*waf.IPSetUpdate{
				&waf.IPSetUpdate{
					Action: aws.String(wafregional.ChangeActionInsert),
					IPSetDescriptor: &waf.IPSetDescriptor{
						Type:  aws.String("IPV4"),
						Value: aws.String("10.0.1.0/24"),
					},
				},
				&waf.IPSetUpdate{
					Action: aws.String(wafregional.ChangeActionInsert),
					IPSetDescriptor: &waf.IPSetDescriptor{
						Type:  aws.String("IPV4"),
						Value: aws.String("10.0.2.0/24"),
					},
				},
				&waf.IPSetUpdate{
					Action: aws.String(wafregional.ChangeActionInsert),
					IPSetDescriptor: &waf.IPSetDescriptor{
						Type:  aws.String("IPV4"),
						Value: aws.String("10.0.3.0/24"),
					},
				},
			},
		},
		{
			// Deletion
			Old: []interface{}{
				map[string]interface{}{"type": "IPV4", "value": "192.0.7.0/24"},
				map[string]interface{}{"type": "IPV4", "value": "192.0.8.0/24"},
			},
			New: []interface{}{},
			ExpectedUpdates: []*waf.IPSetUpdate{
				&waf.IPSetUpdate{
					Action: aws.String(wafregional.ChangeActionDelete),
					IPSetDescriptor: &waf.IPSetDescriptor{
						Type:  aws.String("IPV4"),
						Value: aws.String("192.0.7.0/24"),
					},
				},
				&waf.IPSetUpdate{
					Action: aws.String(wafregional.ChangeActionDelete),
					IPSetDescriptor: &waf.IPSetDescriptor{
						Type:  aws.String("IPV4"),
						Value: aws.String("192.0.8.0/24"),
					},
				},
			},
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			updates := diffWafIpSetDescriptors(tc.Old, tc.New)
			if !reflect.DeepEqual(updates, tc.ExpectedUpdates) {
				t.Fatalf("IPSet updates don't match.\nGiven: %s\nExpected: %s",
					updates, tc.ExpectedUpdates)
			}
		})
	}
}

func testAccCheckAWSWafRegionalIPSetDisappears(v *waf.IPSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).wafregionalconn
		region := testAccProvider.Meta().(*AWSClient).region

		wr := newWafRegionalRetryer(conn, region)
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
  ip_set_descriptor {
    type = "IPV4"
    value = "192.0.7.0/24"
  }
}`, name)
}

func testAccAWSWafRegionalIPSetConfigChangeName(name string) string {
	return fmt.Sprintf(`resource "aws_wafregional_ipset" "ipset" {
  name = "%s"
  ip_set_descriptor {
    type = "IPV4"
    value = "192.0.7.0/24"
  }
}`, name)
}

func testAccAWSWafRegionalIPSetConfigChangeIPSetDescriptors(name string) string {
	return fmt.Sprintf(`resource "aws_wafregional_ipset" "ipset" {
  name = "%s"
  ip_set_descriptor {
    type = "IPV4"
    value = "192.0.8.0/24"
  }
}`, name)
}

func testAccAWSWafRegionalIPSetConfig_noDescriptors(name string) string {
	return fmt.Sprintf(`resource "aws_wafregional_ipset" "ipset" {
   name = "%s"
 }`, name)
}
