package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsAcmCertificate() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsAcmCertificateRead,
		Schema: map[string]*schema.Schema{
			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsAcmCertificateRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).acmconn
	params := &acm.ListCertificatesInput{}
	resp, err := conn.ListCertificates(params)
	if err != nil {
		return errwrap.Wrapf("Error describing certificates: {{err}}", err)
	}

	target := d.Get("domain")
	for _, cert := range resp.CertificateSummaryList {
		if *cert.DomainName == target {
			// Need to call SetId with a value or state won't be written.
			d.SetId(time.Now().UTC().String())
			return d.Set("arn", cert.CertificateArn)
		}
	}

	return fmt.Errorf("No certificate with domain %s found in this region", target)
}
