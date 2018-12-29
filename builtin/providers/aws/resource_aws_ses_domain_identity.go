package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSesDomainIdentity() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSesDomainIdentityCreate,
		Read:   resourceAwsSesDomainIdentityRead,
		Delete: resourceAwsSesDomainIdentityDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"verification_token": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsSesDomainIdentityCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn

	domainName := d.Get("domain").(string)

	createOpts := &ses.VerifyDomainIdentityInput{
		Domain: aws.String(domainName),
	}

	_, err := conn.VerifyDomainIdentity(createOpts)
	if err != nil {
		return fmt.Errorf("Error requesting SES domain identity verification: %s", err)
	}

	d.SetId(domainName)

	return resourceAwsSesDomainIdentityRead(d, meta)
}

func resourceAwsSesDomainIdentityRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn

	domainName := d.Id()
	d.Set("domain", domainName)

	readOpts := &ses.GetIdentityVerificationAttributesInput{
		Identities: []*string{
			aws.String(domainName),
		},
	}

	response, err := conn.GetIdentityVerificationAttributes(readOpts)
	if err != nil {
		log.Printf("[WARN] Error fetching identity verification attributes for %s: %s", d.Id(), err)
		return err
	}

	verificationAttrs, ok := response.VerificationAttributes[domainName]
	if !ok {
		log.Printf("[WARN] Domain not listed in response when fetching verification attributes for %s", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("verification_token", verificationAttrs.VerificationToken)
	return nil
}

func resourceAwsSesDomainIdentityDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn

	domainName := d.Get("domain").(string)

	deleteOpts := &ses.DeleteIdentityInput{
		Identity: aws.String(domainName),
	}

	_, err := conn.DeleteIdentity(deleteOpts)
	if err != nil {
		return fmt.Errorf("Error deleting SES domain identity: %s", err)
	}

	return nil
}
