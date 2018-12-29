package aws

import (
	"fmt"
	"regexp"
	"sort"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestResourceSortByExpirationDate(t *testing.T) {
	certs := []*iam.ServerCertificateMetadata{
		{
			ServerCertificateName: aws.String("oldest"),
			Expiration:            timePtr(time.Now()),
		},
		{
			ServerCertificateName: aws.String("latest"),
			Expiration:            timePtr(time.Now().Add(3 * time.Hour)),
		},
		{
			ServerCertificateName: aws.String("in between"),
			Expiration:            timePtr(time.Now().Add(2 * time.Hour)),
		},
	}
	sort.Sort(certificateByExpiration(certs))
	if *certs[0].ServerCertificateName != "latest" {
		t.Fatalf("Expected first item to be %q, but was %q", "latest", *certs[0].ServerCertificateName)
	}
}

func TestAccAWSDataSourceIAMServerCertificate_basic(t *testing.T) {
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMServerCertificateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIAMServerCertConfig(rInt),
			},
			{
				Config: testAccAwsDataIAMServerCertConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("aws_iam_server_certificate.test_cert", "arn"),
					resource.TestCheckResourceAttrSet("data.aws_iam_server_certificate.test", "arn"),
					resource.TestCheckResourceAttrSet("data.aws_iam_server_certificate.test", "id"),
					resource.TestCheckResourceAttrSet("data.aws_iam_server_certificate.test", "name"),
					resource.TestCheckResourceAttrSet("data.aws_iam_server_certificate.test", "path"),
				),
			},
		},
	})
}

func TestAccAWSDataSourceIAMServerCertificate_matchNamePrefix(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMServerCertificateDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccAwsDataIAMServerCertConfigMatchNamePrefix,
				ExpectError: regexp.MustCompile(`Search for AWS IAM server certificate returned no results`),
			},
		},
	})
}

func testAccAwsDataIAMServerCertConfig(rInt int) string {
	return fmt.Sprintf(`
%s

data "aws_iam_server_certificate" "test" {
  name = "${aws_iam_server_certificate.test_cert.name}"
  latest = true
}
`, testAccIAMServerCertConfig(rInt))
}

var testAccAwsDataIAMServerCertConfigMatchNamePrefix = `
data "aws_iam_server_certificate" "test" {
  name_prefix = "MyCert"
  latest = true
}
`
