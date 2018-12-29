package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSesDomainIdentityVerification() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSesDomainIdentityVerificationCreate,
		Read:   resourceAwsSesDomainIdentityVerificationRead,
		Delete: resourceAwsSesDomainIdentityVerificationDelete,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"domain": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				StateFunc: func(v interface{}) string {
					return strings.TrimSuffix(v.(string), ".")
				},
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(45 * time.Minute),
		},
	}
}

func getAwsSesIdentityVerificationAttributes(conn *ses.SES, domainName string) (*ses.IdentityVerificationAttributes, error) {
	input := &ses.GetIdentityVerificationAttributesInput{
		Identities: []*string{
			aws.String(domainName),
		},
	}

	response, err := conn.GetIdentityVerificationAttributes(input)
	if err != nil {
		return nil, fmt.Errorf("Error getting identity verification attributes: %s", err)
	}

	return response.VerificationAttributes[domainName], nil
}

func resourceAwsSesDomainIdentityVerificationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn
	domainName := strings.TrimSuffix(d.Get("domain").(string), ".")
	err := resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		att, err := getAwsSesIdentityVerificationAttributes(conn, domainName)
		if err != nil {
			return resource.NonRetryableError(fmt.Errorf("Error getting identity verification attributes: %s", err))
		}

		if att == nil {
			return resource.NonRetryableError(fmt.Errorf("SES Domain Identity %s not found in AWS", domainName))
		}

		if aws.StringValue(att.VerificationStatus) != ses.VerificationStatusSuccess {
			return resource.RetryableError(fmt.Errorf("Expected domain verification Success, but was in state %s", aws.StringValue(att.VerificationStatus)))
		}

		return nil
	})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Domain verification successful for %s", domainName)
	d.SetId(domainName)
	return resourceAwsSesDomainIdentityVerificationRead(d, meta)
}

func resourceAwsSesDomainIdentityVerificationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn

	domainName := d.Id()
	d.Set("domain", domainName)

	att, err := getAwsSesIdentityVerificationAttributes(conn, domainName)
	if err != nil {
		log.Printf("[WARN] Error fetching identity verification attributes for %s: %s", d.Id(), err)
		return err
	}

	if att == nil {
		log.Printf("[WARN] Domain not listed in response when fetching verification attributes for %s", d.Id())
		d.SetId("")
		return nil
	}

	if aws.StringValue(att.VerificationStatus) != ses.VerificationStatusSuccess {
		log.Printf("[WARN] Expected domain verification Success, but was %s, tainting verification", aws.StringValue(att.VerificationStatus))
		d.SetId("")
		return nil
	}

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   "ses",
		Region:    meta.(*AWSClient).region,
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("identity/%s", d.Id()),
	}.String()
	d.Set("arn", arn)

	return nil
}

func resourceAwsSesDomainIdentityVerificationDelete(d *schema.ResourceData, meta interface{}) error {
	// No need to do anything, domain identity will be deleted when aws_ses_domain_identity is deleted
	return nil
}
