package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSSMDocument_basic(t *testing.T) {
	name := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMDocumentDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSSMDocumentBasicConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMDocumentExists("aws_ssm_document.foo"),
				),
			},
		},
	})
}

func TestAccAWSSSMDocument_permission(t *testing.T) {
	name := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMDocumentDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSSMDocumentPermissionConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMDocumentExists("aws_ssm_document.foo"),
					resource.TestCheckResourceAttr(
						"aws_ssm_document.foo", "permissions.type", "Share"),
					resource.TestCheckResourceAttr(
						"aws_ssm_document.foo", "permissions.account_ids", "all"),
				),
			},
		},
	})
}

func TestAccAWSSSMDocument_params(t *testing.T) {
	name := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMDocumentDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSSMDocumentParamConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMDocumentExists("aws_ssm_document.foo"),
					resource.TestCheckResourceAttr(
						"aws_ssm_document.foo", "parameter.0.name", "commands"),
					resource.TestCheckResourceAttr(
						"aws_ssm_document.foo", "parameter.0.type", "StringList"),
					resource.TestCheckResourceAttr(
						"aws_ssm_document.foo", "parameter.1.name", "workingDirectory"),
					resource.TestCheckResourceAttr(
						"aws_ssm_document.foo", "parameter.1.type", "String"),
					resource.TestCheckResourceAttr(
						"aws_ssm_document.foo", "parameter.2.name", "executionTimeout"),
					resource.TestCheckResourceAttr(
						"aws_ssm_document.foo", "parameter.2.type", "String"),
				),
			},
		},
	})
}

func testAccCheckAWSSSMDocumentExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SSM Document ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ssmconn

		_, err := conn.DescribeDocument(&ssm.DescribeDocumentInput{
			Name: aws.String(rs.Primary.ID),
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckAWSSSMDocumentDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ssmconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ssm_document" {
			continue
		}

		out, err := conn.DescribeDocument(&ssm.DescribeDocumentInput{
			Name: aws.String(rs.Primary.Attributes["name"]),
		})

		if err != nil {
			// InvalidDocument means it's gone, this is good
			if wserr, ok := err.(awserr.Error); ok && wserr.Code() == "InvalidDocument" {
				return nil
			}
			return err
		}

		if out != nil {
			return fmt.Errorf("Expected AWS SSM Document to be gone, but was still found")
		}

		return nil
	}

	return fmt.Errorf("Default error in SSM Document Test")
}

/*
Based on examples from here: https://docs.aws.amazon.com/AWSEC2/latest/WindowsGuide/create-ssm-doc.html
*/

func testAccAWSSSMDocumentBasicConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_ssm_document" "foo" {
  name = "test_document-%s"

  content = <<DOC
    {
      "schemaVersion": "1.2",
      "description": "Check ip configuration of a Linux instance.",
      "parameters": {

      },
      "runtimeConfig": {
        "aws:runShellScript": {
          "properties": [
            {
              "id": "0.aws:runShellScript",
              "runCommand": ["ifconfig"]
            }
          ]
        }
      }
    }
DOC
}

`, rName)
}

func testAccAWSSSMDocumentPermissionConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_ssm_document" "foo" {
  name = "test_document-%s"

  permissions = {
    type        = "Share"
    account_ids = "all"
  }

  content = <<DOC
    {
      "schemaVersion": "1.2",
      "description": "Check ip configuration of a Linux instance.",
      "parameters": {

      },
      "runtimeConfig": {
        "aws:runShellScript": {
          "properties": [
            {
              "id": "0.aws:runShellScript",
              "runCommand": ["ifconfig"]
            }
          ]
        }
      }
    }
DOC
}
`, rName)
}

func testAccAWSSSMDocumentParamConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_ssm_document" "foo" {
  name = "test_document-%s"

  content = <<DOC
		{
		    "schemaVersion":"1.2",
		    "description":"Run a PowerShell script or specify the paths to scripts to run.",
		    "parameters":{
		        "commands":{
		            "type":"StringList",
		            "description":"(Required) Specify the commands to run or the paths to existing scripts on the instance.",
		            "minItems":1,
		            "displayType":"textarea"
		        },
		        "workingDirectory":{
		            "type":"String",
		            "default":"",
		            "description":"(Optional) The path to the working directory on your instance.",
		            "maxChars":4096
		        },
		        "executionTimeout":{
		            "type":"String",
		            "default":"3600",
		            "description":"(Optional) The time in seconds for a command to be completed before it is considered to have failed. Default is 3600 (1 hour). Maximum is 28800 (8 hours).",
		            "allowedPattern":"([1-9][0-9]{0,3})|(1[0-9]{1,4})|(2[0-7][0-9]{1,3})|(28[0-7][0-9]{1,2})|(28800)"
		        }
		    },
		    "runtimeConfig":{
		        "aws:runPowerShellScript":{
		            "properties":[
		                {
		                    "id":"0.aws:runPowerShellScript",
		                    "runCommand":"{{ commands }}",
		                    "workingDirectory":"{{ workingDirectory }}",
		                    "timeoutSeconds":"{{ executionTimeout }}"
		                }
		            ]
		        }
		    }
		}
DOC
}

`, rName)
}
