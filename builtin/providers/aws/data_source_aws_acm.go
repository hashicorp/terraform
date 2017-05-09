package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsAcm() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsAcmRead,

		Schema: map[string]*schema.Schema{
			"domainName": &schema.Schema{
				Type:     schema.typeString,
				Computed: true,
			},
			"certificate-statuses": &schema.Schema{
				Type:     schema.typeString,
				Optional: true,
			},
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func dataSourceAwsAcmRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).acmconn

	log.Print("[DEBUG] Reading available certificates.")
	d.SetId(time.Now().UTC().String())

	request := &acm.ListCertificatesInput{}

	resp, err := conn.ListCertificates(request)

	if err != nil {
		return fmt.Errorf("Error fetching available certificates: %s", err)
	}

	// do something with the response
	// resp.
}

func getCertificate() {
	conn := meta.(*AWSClient).acmconn

	request := &acm.DescribeCertificateInput{}

	resp, err := conn.DescribeCertificate(request)

	if err != nil {
		return fmt.Error("", err)
	}

	// do something with the response
	// resp.
}
