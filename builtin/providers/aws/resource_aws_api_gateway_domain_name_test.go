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

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayDomainNameDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAPIGatewayDomainNameConfigCreate(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayDomainNameExists("aws_api_gateway_domain_name.test", &conf),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_domain_name.test", "certificate_body", certificate_body),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_domain_name.test", "certificate_chain", certificate_chain),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_domain_name.test", "certificate_name", "Example"),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_domain_name.test", "certificate_private_key", certificate_private_key),
					resource.TestCheckResourceAttr(
						"aws_api_gateway_domain_name.test", "domain_name", "test-api-gateway-domain-cert0.com"),
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

// Expires in 10 years: July 18, 2026
const certificate_body = `-----BEGIN CERTIFICATE-----
MIIEKzCCAxOgAwIBAgIQZosNkfHTE1ESrtg6tyEbGDANBgkqhkiG9w0BAQsFADCB
yjELMAkGA1UEBhMCVVMxCzAJBgNVBAgTAkNBMRYwFAYDVQQHEw1QaXJhdGUgSGFy
Ym9yMRkwFwYDVQQJExA1ODc5IENvdHRvbiBMaW5rMRMwEQYDVQQREwo5NTU1OS0x
MjI3MRUwEwYDVQQKEwxFeGFtcGxlLCBJbmMxKDAmBgNVBAsTH0RlcGFydG1lbnQg
b2YgVGVycmFmb3JtIFRlc3RpbmcxGTAXBgNVBAMTEHRlc3QtQ0EtY2VydC5jb20x
CjAIBgNVBAUTATIwHhcNMTYwNzE5MDAyMzQxWhcNMjYwNzE3MDAyMzQxWjB6MQkw
BwYDVQQGEwAxCTAHBgNVBAgTADEJMAcGA1UEBxMAMQkwBwYDVQQREwAxFTATBgNV
BAoTDEV4YW1wbGUsIEluYzEJMAcGA1UECxMAMSowKAYDVQQDEyF0ZXN0LWFwaS1n
YXRld2F5LWRvbWFpbi1jZXJ0MC5jb20wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAw
ggEKAoIBAQDGyA7eZ70OVBQiPdG48ThHFwAaxxSZTmNMUsOyAtYbyN/+YijCVeJN
FadfrcRz3Bd/byJuv8+M4C9aBLWO5M4E32UCWABn6Carg8ZpBHnLXqcg8UUnsW06
hkGirP1mKbgH24ABuxvry4r3tFS6UhM4MYCZshjLKMd5AdYRvKUxXc5yf9aYc3uH
m1QfjJXbEuVqBoVWhzCKLpDRnsUPlN8/0QX+sGVcYTecZtwkmq6JvyAxYO+/KV/z
vRJFuMnJiBrmvMTuG2weVbtbj8pWoP7YLxufMiTgtVEDb0fxZaCZSnsu2owCNyiI
bwgb+/YS3oeUDrNQwMnU6DXErD/jRFEHAgMBAAGjXDBaMA4GA1UdDwEB/wQEAwIF
oDAZBgNVHSUEEjAQBggrBgEFBQcDAQYEVR0lADAMBgNVHRMBAf8EAjAAMB8GA1Ud
IwQYMBaAFJb+4Yye0eUswIfcnAVearIegOeIMA0GCSqGSIb3DQEBCwUAA4IBAQDC
sVKfshMvUeqngwxkKu9vETxD1bdDpRujtjVSFyJLpDfjzU/R8VlkGQPnSC6kIKef
DdGzlXDCtVP/UCspnPqYi3q/d6K0ohSAmmvD4MbmLO1oWYTx394iDjBSkexhtP7e
c8k8x81KEhkpVPOZR9pSSnrTWshGE02WnX0PQliGt8bj/n9sK95qYKVS/lmLrMmr
CIE8uSCDNTBQ+eidLzq7EHK5v592eu54YitBb36MXGZK7QvWYZA/x9g/ws5lbAHh
6VWEKdtCOsJE+bNDVLH8Lppv7mGraOPAkXDZGQ+t/CsZiAy+jxGwNcCrYgYkGs/1
wDz6uFIEEUhEFN820dq2
-----END CERTIFICATE-----
`

// Expires in 10 years: July 18, 2026
const certificate_chain = `-----BEGIN CERTIFICATE-----
MIIEnjCCA4agAwIBAgIQRwdu33h5xkb8WeZlnNzSrzANBgkqhkiG9w0BAQsFADCB
yjELMAkGA1UEBhMCVVMxCzAJBgNVBAgTAkNBMRYwFAYDVQQHEw1QaXJhdGUgSGFy
Ym9yMRkwFwYDVQQJExA1ODc5IENvdHRvbiBMaW5rMRMwEQYDVQQREwo5NTU1OS0x
MjI3MRUwEwYDVQQKEwxFeGFtcGxlLCBJbmMxKDAmBgNVBAsTH0RlcGFydG1lbnQg
b2YgVGVycmFmb3JtIFRlc3RpbmcxGTAXBgNVBAMTEHRlc3QtQ0EtY2VydC5jb20x
CjAIBgNVBAUTATIwHhcNMTYwNzE5MDAyMzQxWhcNMjYwNzE3MDAyMzQxWjCByjEL
MAkGA1UEBhMCVVMxCzAJBgNVBAgTAkNBMRYwFAYDVQQHEw1QaXJhdGUgSGFyYm9y
MRkwFwYDVQQJExA1ODc5IENvdHRvbiBMaW5rMRMwEQYDVQQREwo5NTU1OS0xMjI3
MRUwEwYDVQQKEwxFeGFtcGxlLCBJbmMxKDAmBgNVBAsTH0RlcGFydG1lbnQgb2Yg
VGVycmFmb3JtIFRlc3RpbmcxGTAXBgNVBAMTEHRlc3QtQ0EtY2VydC5jb20xCjAI
BgNVBAUTATIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDH/r1+t19l
Ftg87724PH1ibH73jjue6Iv2xvnh2Mvl3v6jKf6Mfp5VZitB2adZ5nPRwNmD6neT
e4vkGk99b3w66SLgdbvIQY50A/Bnl1sOU/R0mADafuQCfBKBBJ51mTEWhbqXnD9i
YfHGtUzfUMqaG+XtRhPCEpyxq7IkR4zYdOjcsLpclN/ybviI9v/Vr6RSXAAwmGZa
6g3RdVsL9wsuQIybPfEjhzbi6CXI9iWTLsT+Aul5cXcrXpTjq2qbVG9Hi1qbvFdP
/JhgN6eixAGKLR+rhNZHTzk9Hf6aHbWfqYNqK6jTqoxK7V7ADOwHwGfCJR4UsF/5
XkTNypg8Ln6rAgMBAAGjfjB8MA4GA1UdDwEB/wQEAwICBDAZBgNVHSUEEjAQBggr
BgEFBQcDAQYEVR0lADAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBSW/uGMntHl
LMCH3JwFXmqyHoDniDAfBgNVHSMEGDAWgBSW/uGMntHlLMCH3JwFXmqyHoDniDAN
BgkqhkiG9w0BAQsFAAOCAQEAisd6toirkcbYjL6Z9wuZxWwKwqFGquqGtSx60UVx
lLlP97mTOM7dRkwfNMhWm9tlWNsiym3Ru8rDZPBypDPMvyanp+YvWj/JBVEl/I3/
nN5QxmCIFxBHFdQ2+woekhGj/dqqQOEjtdF5079djS1oOL2hX/akGprI64/Yc0uD
qW166H77fVTYrdH6TBBcWvr2K01gHFygwlYXOoojSzr4uKvmhkaz3d5Vjt2xOWoz
iikRD1dMLZ3bGoWkccZQBFTb5qTCe2vVtjpBpqfsD3BIaiHYKAVUkTJQFLzFGNFm
EhciXQzvTF6/JBtgWI8Nl5ISzfBrtvJAQjjYkIMizPv7Dg==
-----END CERTIFICATE-----
`

// Expires in 10 years: July 18, 2026
const certificate_private_key = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAxsgO3me9DlQUIj3RuPE4RxcAGscUmU5jTFLDsgLWG8jf/mIo
wlXiTRWnX63Ec9wXf28ibr/PjOAvWgS1juTOBN9lAlgAZ+gmq4PGaQR5y16nIPFF
J7FtOoZBoqz9Zim4B9uAAbsb68uK97RUulITODGAmbIYyyjHeQHWEbylMV3Ocn/W
mHN7h5tUH4yV2xLlagaFVocwii6Q0Z7FD5TfP9EF/rBlXGE3nGbcJJquib8gMWDv
vylf870SRbjJyYga5rzE7htsHlW7W4/KVqD+2C8bnzIk4LVRA29H8WWgmUp7LtqM
AjcoiG8IG/v2Et6HlA6zUMDJ1Og1xKw/40RRBwIDAQABAoIBABwKirZrEetchv6R
k+0v8g1tPDGK1egOe8l/f2W0Krn+q0J6XF+Vt/fBzzubCrSBXrs2VTgkTMYFtghP
08DVnA5p6RjciyodQJ8/VpTn8bpznsXx4xyHVe5ElCu7lX988R4Co9sapwSrUO5C
fRVPkLCDoy2LRx4ZoZH7ZVRZNUBySH2HXTwi03u9eZFrxuU+JxD9kJ11d/bNh4yx
F7/6P6hAVGm3oR3kegvp5fWkxS6MfIxSrZoHVRLiEu2NxGc7AqC+naXwVUx8FwKS
dtn7d0lJFE/ntAp2dSNPBTk6c0XG908DSZLjuWIl3K92BcjIUWPyuXCzr37BC81g
Q7su8HECgYEA1s4Y9pcOsqtZ3eA8uEU2yfDoediHq8RJiaoplC7JQzOGRMTVGXeX
cK/ya5QVncrwM7YyIJM51DLasxhBHl2Cz3dQ8940hKMTGriM/gu8MlR+Ie9cjk0B
G9dfUo+v5Ne1jugbbJZ65IjXZYMAQS9MjfCIYHhprtIEuJDM2s44A68CgYEA7OdH
Xp7zprO+1iurdRuo3v4nRtw6IMZL3HoKisKOpRbJgRDDNc9Pvv5KBWoXuErnib3b
DBJLx9YQQqqjOmqf7FMdVJrhv0ebQGS4tryjW8PXJOmRowh9QLnAQAddRpAltwcf
6blJ5UzJe6C+pFXYv2Xh1Jd1JPe98+2szRNsZikCgYAgU74QBmXQ39bTfHbG6Ku5
ModaJwsr/4ttq208ftoNQgjX+qNzhLsG24PpSs0CBVOnBKmAm4eddtXRFDpgnoQc
QwGs4ekXeQ9b+yBE73EwReUBqGtOgypCjWQsIbHAB/KsAiR2cCMol6uK/G8iYELu
LZ/onNaS18qcGDasS1LEwwKBgQCLJBvK+1jn5FKFwAhoM+Kvdl7jQ53wegc8a4Gd
lj/pvsSDRbEh/a085GXdYD6mQ3hScmwhXu2bZaMPROGyAcYEK5zigEVu70PEQmQr
EAhycUf/qh+bvfSy+2ZrNOgX9bnxEgIwaF96iesc7YCLTNCNOe21y29GUywCBOql
WG8mYQKBgBpYt8QokGkQCFub5X1RhIp/qnzaxD4gbh1xA7eDYcRUtkf7waS2h+Rd
bum30SIUTM/EH8h41MLWFbmRtaCsSM0kRWlV6G8xdkMLEE4dv4DeDSEgtURtSG8Z
N8InFqoE91SzYufu3gz9zxc0kAe2kVG/kjq67vZtZ4I1xm5yocc4
-----END RSA PRIVATE KEY-----
`

func testAccAWSAPIGatewayDomainNameConfigCreate() string {
	return fmt.Sprintf(`
resource "aws_api_gateway_domain_name" "test" {
  certificate_body = "%v"
  certificate_chain = "%v"
  certificate_name = "Example"
  certificate_private_key = "%v"
  domain_name = "test-api-gateway-domain-cert0.com"
}
`, certificate_body, certificate_chain, certificate_private_key)
}
