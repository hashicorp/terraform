package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"log"
)

func resourceAwsOrganization() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsOrganizationCreate,
		Read:   resourceAwsOrganizationRead,
		Update: resourceAwsOrganizationUpdate,
		Delete: resourceAwsOrganizationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"feature_set": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "ALL",
				ValidateFunc: validation.StringInSlice([]string{"ALL", "CONSOLIDATED_BILLING"}, true),
			},
		},
	}
}

func resourceAwsOrganizationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).orgsconn

	// Create the organization
	createOpts := &organizations.CreateOrganizationInput{
		FeatureSet: aws.String(d.Get("feature_set").(string)),
	}
	log.Printf("[DEBUG] Organization create config: %#v", createOpts)

	resp, err := conn.CreateOrganization(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating organization: %s", err)
	}

	// Get the ID and store it
	org := resp.Organization
	d.SetId(*org.Id)
	log.Printf("[INFO] Organization ID: %s", d.Id())

	return resourceAwsOrganizationUpdate(d, meta)
}

func resourceAwsOrganizationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).orgsconn
	org, err := conn.DescribeOrganization(&organizations.DescribeOrganizationInput{})
	if err != nil {
		if orgerr, ok := err.(awserr.Error); ok && orgerr.Code() == "AWSOrganizationsNotInUseException" {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("arn", org.Organization.Arn)
	d.Set("feature_set", org.Organization.FeatureSet)
	d.Set("master_account_arn", org.Organization.MasterAccountArn)
	d.Set("master_account_email", org.Organization.MasterAccountEmail)
	d.Set("master_account_id", org.Organization.MasterAccountId)
	return nil
}

func resourceAwsOrganizationUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceAwsOrganizationRead(d, meta)
}

func resourceAwsOrganizationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).orgsconn
	_, err := conn.DeleteOrganization(&organizations.DeleteOrganizationInput{})
	if err != nil {
		return err
	}

	return nil

}
