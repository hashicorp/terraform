package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAPIGatewayDomainName_basic(t *testing.T) {
	var conf apigateway.DomainName

	// Our test cert is for a wildcard on this domain
	uniqueId := resource.UniqueId()
	name := fmt.Sprintf("%s.tf-acc.invalid", uniqueId)
	nameModified := fmt.Sprintf("test-acc.%s.tf-acc.invalid", uniqueId)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayDomainNameDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAPIGatewayDomainNameConfigCreate(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayDomainNameExists("aws_api_gateway_domain_name.test", &conf),
					resource.TestCheckResourceAttr("aws_api_gateway_domain_name.test", "certificate_body", testAccAWSAPIGatewayCertBody),
					resource.TestCheckResourceAttr("aws_api_gateway_domain_name.test", "certificate_chain", testAccAWSAPIGatewayCertChain),
					resource.TestCheckResourceAttr("aws_api_gateway_domain_name.test", "certificate_name", "tf-acc-apigateway-domain-name"),
					resource.TestCheckResourceAttr("aws_api_gateway_domain_name.test", "certificate_private_key", testAccAWSAPIGatewayCertPrivateKey),
					resource.TestCheckResourceAttr("aws_api_gateway_domain_name.test", "domain_name", name),
					resource.TestCheckResourceAttrSet("aws_api_gateway_domain_name.test", "certificate_upload_date"),
				),
			},
			{
				Config: testAccAWSAPIGatewayDomainNameConfigUpdate(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayDomainNameExists("aws_api_gateway_domain_name.test", &conf),
					resource.TestCheckResourceAttr("aws_api_gateway_domain_name.test", "certificate_body", testAccAWSAPIGatewayCertBody),
					resource.TestCheckResourceAttr("aws_api_gateway_domain_name.test", "certificate_chain", testAccAWSAPIGatewayCertChain),
					resource.TestCheckResourceAttr("aws_api_gateway_domain_name.test", "certificate_name", "tf-acc-apigateway-domain-name"),
					resource.TestCheckResourceAttr("aws_api_gateway_domain_name.test", "certificate_private_key", testAccAWSAPIGatewayCertPrivateKey),
					resource.TestCheckResourceAttr("aws_api_gateway_domain_name.test", "domain_name", nameModified),
					resource.TestCheckResourceAttrSet("aws_api_gateway_domain_name.test", "certificate_upload_date"),
				),
			},
		},
	})
}

func testAccCheckAWSAPIGatewayDomainNameExists(n string, res *apigateway.DomainName) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No API Gateway DomainName ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).apigateway

		req := &apigateway.GetDomainNameInput{
			DomainName: aws.String(rs.Primary.ID),
		}
		describe, err := conn.GetDomainName(req)
		if err != nil {
			return err
		}

		if *describe.DomainName != rs.Primary.ID {
			return fmt.Errorf("APIGateway DomainName not found")
		}

		*res = *describe

		return nil
	}
}

func testAccCheckAWSAPIGatewayDomainNameDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).apigateway

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_api_gateway_api_key" {
			continue
		}

		describe, err := conn.GetDomainNames(&apigateway.GetDomainNamesInput{})

		if err == nil {
			if len(describe.Items) != 0 &&
				*describe.Items[0].DomainName == rs.Primary.ID {
				return fmt.Errorf("API Gateway DomainName still exists")
			}
		}

		aws2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if aws2err.Code() != "NotFoundException" {
			return err
		}

		return nil
	}

	return nil
}

// Expires on August 20, 2026
const testAccAWSAPIGatewayCertBody = `-----BEGIN CERTIFICATE-----
MIIEfjCCA2agAwIBAgIRAPHiUbRWC/Ff/reiyzqh3wAwDQYJKoZIhvcNAQELBQAw
gcsxCzAJBgNVBAYTAlVTMQswCQYDVQQIEwJDQTEWMBQGA1UEBxMNUGlyYXRlIEhh
cmJvcjEZMBcGA1UECRMQNTg3OSBDb3R0b24gTGluazETMBEGA1UEERMKOTU1NTkt
MTIyNzEVMBMGA1UEChMMRXhhbXBsZSwgSW5jMSgwJgYDVQQLEx9EZXBhcnRtZW50
IG9mIFRlcnJhZm9ybSBUZXN0aW5nMRowGAYDVQQDExF0Zi1hY2MtY2EuaW52YWxp
ZDEKMAgGA1UEBRMBMjAeFw0xNjA4MjIxNjQ1NTZaFw0yNjA4MjAxNjQ1NTZaMIHK
MQswCQYDVQQGEwJVUzELMAkGA1UECBMCQ0ExFjAUBgNVBAcTDVBpcmF0ZSBIYXJi
b3IxGTAXBgNVBAkTEDU4NzkgQ290dG9uIExpbmsxEzARBgNVBBETCjk1NTU5LTEy
MjcxFTATBgNVBAoTDEV4YW1wbGUsIEluYzEoMCYGA1UECxMfRGVwYXJ0bWVudCBv
ZiBUZXJyYWZvcm0gVGVzdGluZzEZMBcGA1UEAxMQKi50Zi1hY2MuaW52YWxpZDEK
MAgGA1UEBRMBMjCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAL/xTPoz
3tnvkLjthqsMKwWVtE1yi210UnseBYkc2w6iQUXMCsXkw2PZXW6QIW7o1GNjnrE5
9jkeEJ2ABEYyor+ooaNZmjJ0dvY1tFerMhsOkQudDBz+eaR13UQvMFc42nFWrS5/
XCInm4HjWblEfw2b/wbGwTPzIN3+zWdt2XTsSzIRUYgjXy/93aLIVhWaYDsP45RM
C48qhJKSsYOVH/PSkgw4AfohDXzn2iltTUBFdGZqBufMW4sPp8G8DIOvt01NsXB2
QLLVgcMJVegx9xxg50+cf6ILck5Ap/SzkCN7XYk8zWC36/QIkiQ6bMMy0nJtgE0B
jwmwnFF+zF/A4oMCAwEAAaNcMFowDgYDVR0PAQH/BAQDAgWgMBkGA1UdJQQSMBAG
CCsGAQUFBwMBBgRVHSUAMAwGA1UdEwEB/wQCMAAwHwYDVR0jBBgwFoAU/hWQ+UUh
7EiLd0fCsgy1jvSgxkswDQYJKoZIhvcNAQELBQADggEBABnTdSEOqj2+7FTeiSR9
iPGQ3swhrmk59j8XUszrlm6MvZX9AeV9gLcE9J0PfFaAcZM0pR+XZBmyZLoBxUgU
8Y17yWwAxZo13EwgzTOmebDxm4W+Og4vBxsKS3x/L8D8BKdzBcmVJvggJqTs5Qau
ePetAQcD/Tfbq7vY9fNQJ4XQTlF5sN625miijTQcNg/s/H2Fj3FENR3UaFiQDtLQ
eatV0KGF/ik8XFHPbFm7XUZfUcDxvoaljglJfXyVttU956j892ZakvAHzhylFRyy
/wJnKV7K3zQVCEV0n/dJpXhf7w9PuWHrCp+HZJLMPMtm1IPblVxPNwHhGBG40I+Z
FSg=
-----END CERTIFICATE-----
`

// Expires on August 20, 2026
const testAccAWSAPIGatewayCertChain = `-----BEGIN CERTIFICATE-----
MIIEoDCCA4igAwIBAgIQRR1EBw7bN+mMYtXrrvzLdzANBgkqhkiG9w0BAQsFADCB
yzELMAkGA1UEBhMCVVMxCzAJBgNVBAgTAkNBMRYwFAYDVQQHEw1QaXJhdGUgSGFy
Ym9yMRkwFwYDVQQJExA1ODc5IENvdHRvbiBMaW5rMRMwEQYDVQQREwo5NTU1OS0x
MjI3MRUwEwYDVQQKEwxFeGFtcGxlLCBJbmMxKDAmBgNVBAsTH0RlcGFydG1lbnQg
b2YgVGVycmFmb3JtIFRlc3RpbmcxGjAYBgNVBAMTEXRmLWFjYy1jYS5pbnZhbGlk
MQowCAYDVQQFEwEyMB4XDTE2MDgyMjE2NDU1NloXDTI2MDgyMDE2NDU1Nlowgcsx
CzAJBgNVBAYTAlVTMQswCQYDVQQIEwJDQTEWMBQGA1UEBxMNUGlyYXRlIEhhcmJv
cjEZMBcGA1UECRMQNTg3OSBDb3R0b24gTGluazETMBEGA1UEERMKOTU1NTktMTIy
NzEVMBMGA1UEChMMRXhhbXBsZSwgSW5jMSgwJgYDVQQLEx9EZXBhcnRtZW50IG9m
IFRlcnJhZm9ybSBUZXN0aW5nMRowGAYDVQQDExF0Zi1hY2MtY2EuaW52YWxpZDEK
MAgGA1UEBRMBMjCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAK7k1xdX
HdA0HTajtSPvCwIeNE4HO3IsYn9jnUBvhQ52EJ72GPzlg91AhdVufIncJRjk51iP
n86TO/bg/dx4d+G45aUFw7WmZGTTZjurhnVWCDPeeetItV3PmsWHkwTiBj4GJYW4
1mJk0ACjmmGEyz24T46Cn87Tljk+ivdCQL/oYTjpC7jc2i5b7ziiwsToWMEljWV1
cf/4qQNFmFnETbaQW2qcFdjUoO8N1gRNkC9yvOjo6Fc+lETRHn3e926jDSaqfbwJ
hKKBoXciYnddNXgA83GBTdnsRiRoD4zFEhNThmgvy5MmkEEdwKl5QIz6ajo4Xxtd
54XZpt0y/ZVy6BECAwEAAaN+MHwwDgYDVR0PAQH/BAQDAgIEMBkGA1UdJQQSMBAG
CCsGAQUFBwMBBgRVHSUAMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFP4VkPlF
IexIi3dHwrIMtY70oMZLMB8GA1UdIwQYMBaAFP4VkPlFIexIi3dHwrIMtY70oMZL
MA0GCSqGSIb3DQEBCwUAA4IBAQBlXSOkvE/c8pcI6vijInnAWY+eX2W7njCzEfmP
gkDeW/2A1/6eSM6C/ym+9zEgZ9YLxPfiQKzN8Ej+EziFiZW3ZBHnWcW9rZt36wFA
swF2nIqWa5nkVnAX2aPVfULN1u1BQiv3XZqNNlBk6m9AsgxiZ9jtOY8oQQU3uDYe
bs9ttLMKJApbe08sKmPawVlXHqj6qb85Pg4bzzknA+euRee9KIi87eTlWc/Ht+WN
Huh16WqokePii7e/PAaiTAe7ubsPav29lzmHq2NooXrs+FljXveQJpBqu6wKKTXe
6/5BBsp2474RiZgTbuO2Mbr2nwcoJ9V6iu1M9siCGbbt1nQr
-----END CERTIFICATE-----
`

const testAccAWSAPIGatewayCertPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAv/FM+jPe2e+QuO2GqwwrBZW0TXKLbXRSex4FiRzbDqJBRcwK
xeTDY9ldbpAhbujUY2OesTn2OR4QnYAERjKiv6iho1maMnR29jW0V6syGw6RC50M
HP55pHXdRC8wVzjacVatLn9cIiebgeNZuUR/DZv/BsbBM/Mg3f7NZ23ZdOxLMhFR
iCNfL/3doshWFZpgOw/jlEwLjyqEkpKxg5Uf89KSDDgB+iENfOfaKW1NQEV0ZmoG
58xbiw+nwbwMg6+3TU2xcHZAstWBwwlV6DH3HGDnT5x/ogtyTkCn9LOQI3tdiTzN
YLfr9AiSJDpswzLScm2ATQGPCbCcUX7MX8DigwIDAQABAoIBAFSRcXQPpJFrDt2b
sajtTItCYVV6MVpBVRHvsUqvDwkMjiu9ccWtPDVjENpk4IYoSWOdAc9eFVEnIPTz
8W4oYzKEjusU0G6Ih92E3fd+cy4epeNzB2JC8L94OswO6oKThxNGuDjzXlmiD88T
p3WMa1pIr/2BVqCX75Q/7qoyaQwtR/anvOes154gRGY70ucsUpEzo3fyfXdCKO/o
y1quQ4mmCS50lPCJ3i8JZH2YmGY+xThnXzk7Q9ipjGrHTjinpX8f9tQB2RRwqlA/
uQ+/q6cOKUAehLtriXyr2QAoCzdz1Lfm8L5KQ3EdrxUUe9dhs/a7bKw4N5eHjw/x
eBUagsECgYEAyoaIqZ0ur5NQEHuY9Givs6a+ObWLjM18EdfQ7zX7NJRPL3q0jdkm
KW+pn2OzoRxhwgaD6WuJJmj3rd3ledkwxoO6Efwo8x2z1u/T43FsknW/X3wJLLga
fPU2T9lSHRufBQUAhPYkf4dSqUklHJ4YSMafbCkvSXKGZqGVLQfu1iECgYEA8p9s
/5+d0Pcnykau7Hp8a9z9ehHEIF+ce/wb6/6p3fgCP04FLvKqs74p9SrEnyl9uiFx
ZC9P2XhmEwNFGA+qKPaRBo4RoGeAYgCKD0BfdaGfmp6WLhO0pSnl4c06bXnvwqbD
DhhT6UvMs6m/uSufwtfCGO48yYiY1z6DqBjuHCMCgYBWJyrltHbSu8D4cgucFRiB
PPJ5DDCkIhmgYYWA7R7CvEB/OxyppvFj+RtYMYqNg8xWRH1DA7rhOw/5x4ZB8lGc
cRbrZbBp033YdkdV3r9IAoz5aoNgoaSq+Yk0KIeU2FYqRXl2FltqYL+aQgJmjR5Z
fxz8Xvy9qtlfuWcDM/e24QKBgQDGtH8mk+lCfUkfRuh4UJCaHnGSif5grS2R9ZZA
n18rpbThd9qS6reXYgUm/5Hs8KRBzqX5cS4qY4rlw2XRIPMxfU6lWbFh96KToPFx
MD1+L5JxpbRFpGnsYvYdCmHxy03r03wojRAcH7JU6o9U7j936hDTLjqmq7LRhid5
goFwlQKBgQCGxTWqQu9FJjtmjdM/ARi7CcwWT40NI8ybyebjDtElekILFMUwO+/w
M78Me3d5g/oyeS6qCfyK1f7vBb/WhxLBBegaqMGUWbRoKYD0LH5U3zkqZ9ehnDFB
vhcEnoxPD9HmdFkRduJscyzVNJ50lMuQ3uk7zwiYuS/Z7shH81xzMQ==
-----END RSA PRIVATE KEY-----
`

func testAccAWSAPIGatewayDomainNameConfigCreate(name string) string {
	return fmt.Sprintf(`
resource "aws_api_gateway_domain_name" "test" {
  domain_name = "%s"
  certificate_body = <<EOF
%vEOF
  certificate_chain = <<EOF
%vEOF
  certificate_name = "tf-acc-apigateway-domain-name"
  certificate_private_key = <<EOF
%vEOF
}
`, name, testAccAWSAPIGatewayCertBody, testAccAWSAPIGatewayCertChain, testAccAWSAPIGatewayCertPrivateKey)
}

func testAccAWSAPIGatewayDomainNameConfigUpdate(name string) string {
	return fmt.Sprintf(`
resource "aws_api_gateway_domain_name" "test" {
  domain_name = "test-acc.%s"
  certificate_body = <<EOF
%vEOF
  certificate_chain = <<EOF
%vEOF
  certificate_name = "tf-acc-apigateway-domain-name"
  certificate_private_key = <<EOF
%vEOF
}
`, name, testAccAWSAPIGatewayCertBody, testAccAWSAPIGatewayCertChain, testAccAWSAPIGatewayCertPrivateKey)
}
