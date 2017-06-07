package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsAcmCertificate() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsAcmCertificateRead,
		Schema: map[string]*schema.Schema{
			"domain": {
				Type:     schema.TypeString,
				Required: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"statuses": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"types": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceAwsAcmCertificateRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).acmconn
	params := &acm.ListCertificatesInput{}

	target := d.Get("domain")

	statuses, ok := d.GetOk("statuses")
	if ok {
		statusStrings := statuses.([]interface{})
		params.CertificateStatuses = expandStringList(statusStrings)
	} else {
		params.CertificateStatuses = []*string{aws.String("ISSUED")}
	}

	var arns []string
	err := conn.ListCertificatesPages(params, func(page *acm.ListCertificatesOutput, lastPage bool) bool {
		for _, cert := range page.CertificateSummaryList {
			if *cert.DomainName == target {
				arns = append(arns, *cert.CertificateArn)
			}
		}

		return true
	})
	if err != nil {
		return errwrap.Wrapf("Error describing certificates: {{err}}", err)
	}

	// filter based on certificate type (imported or aws-issued)
	types, ok := d.GetOk("types")
	if ok {
		typesStrings := expandStringList(types.([]interface{}))
		var matchedArns []string
		for _, arn := range arns {
			params := &acm.DescribeCertificateInput{}
			params.CertificateArn = &arn

			description, err := conn.DescribeCertificate(params)
			if err != nil {
				return errwrap.Wrapf("Error describing certificates: {{err}}", err)
			}

			for _, certType := range typesStrings {
				if *description.Certificate.Type == *certType {
					matchedArns = append(matchedArns, arn)
					break
				}
			}
		}

		arns = matchedArns
	}

	if len(arns) == 0 {
		return fmt.Errorf("No certificate for domain %q found in this region.", target)
	}
	if len(arns) > 1 {
		return fmt.Errorf("Multiple certificates for domain %q found in this region.", target)
	}

	d.SetId(time.Now().UTC().String())
	d.Set("arn", arns[0])

	return nil
}
