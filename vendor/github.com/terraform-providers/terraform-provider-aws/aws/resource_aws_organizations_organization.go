package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsOrganizationsOrganization() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsOrganizationsOrganizationCreate,
		Read:   resourceAwsOrganizationsOrganizationRead,
		Update: resourceAwsOrganizationsOrganizationUpdate,
		Delete: resourceAwsOrganizationsOrganizationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"master_account_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"master_account_email": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"master_account_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"aws_service_access_principals": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"feature_set": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  organizations.OrganizationFeatureSetAll,
				ValidateFunc: validation.StringInSlice([]string{
					organizations.OrganizationFeatureSetAll,
					organizations.OrganizationFeatureSetConsolidatedBilling,
				}, true),
			},
		},
	}
}

func resourceAwsOrganizationsOrganizationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).organizationsconn

	createOpts := &organizations.CreateOrganizationInput{
		FeatureSet: aws.String(d.Get("feature_set").(string)),
	}
	log.Printf("[DEBUG] Creating Organization: %#v", createOpts)

	resp, err := conn.CreateOrganization(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating organization: %s", err)
	}

	org := resp.Organization
	d.SetId(*org.Id)

	awsServiceAccessPrincipals := d.Get("aws_service_access_principals").(*schema.Set).List()
	for _, principalRaw := range awsServiceAccessPrincipals {
		principal := principalRaw.(string)
		input := &organizations.EnableAWSServiceAccessInput{
			ServicePrincipal: aws.String(principal),
		}

		log.Printf("[DEBUG] Enabling AWS Service Access in Organization: %s", input)
		_, err := conn.EnableAWSServiceAccess(input)

		if err != nil {
			return fmt.Errorf("error enabling AWS Service Access (%s) in Organization: %s", principal, err)
		}
	}

	return resourceAwsOrganizationsOrganizationRead(d, meta)
}

func resourceAwsOrganizationsOrganizationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).organizationsconn

	log.Printf("[INFO] Reading Organization: %s", d.Id())
	org, err := conn.DescribeOrganization(&organizations.DescribeOrganizationInput{})

	if isAWSErr(err, organizations.ErrCodeAWSOrganizationsNotInUseException, "") {
		log.Printf("[WARN] Organization does not exist, removing from state: %s", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error describing Organization: %s", err)
	}

	d.Set("arn", org.Organization.Arn)
	d.Set("feature_set", org.Organization.FeatureSet)
	d.Set("master_account_arn", org.Organization.MasterAccountArn)
	d.Set("master_account_email", org.Organization.MasterAccountEmail)
	d.Set("master_account_id", org.Organization.MasterAccountId)

	awsServiceAccessPrincipals := make([]string, 0)

	// ConstraintViolationException: The request failed because the organization does not have all features enabled. Please enable all features in your organization and then retry.
	if aws.StringValue(org.Organization.FeatureSet) == organizations.OrganizationFeatureSetAll {
		err = conn.ListAWSServiceAccessForOrganizationPages(&organizations.ListAWSServiceAccessForOrganizationInput{}, func(page *organizations.ListAWSServiceAccessForOrganizationOutput, lastPage bool) bool {
			for _, enabledServicePrincipal := range page.EnabledServicePrincipals {
				awsServiceAccessPrincipals = append(awsServiceAccessPrincipals, aws.StringValue(enabledServicePrincipal.ServicePrincipal))
			}
			return !lastPage
		})

		if err != nil {
			return fmt.Errorf("error listing AWS Service Access for Organization (%s): %s", d.Id(), err)
		}
	}

	if err := d.Set("aws_service_access_principals", awsServiceAccessPrincipals); err != nil {
		return fmt.Errorf("error setting aws_service_access_principals: %s", err)
	}

	return nil
}

func resourceAwsOrganizationsOrganizationUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).organizationsconn

	if d.HasChange("aws_service_access_principals") {
		oldRaw, newRaw := d.GetChange("aws_service_access_principals")
		oldSet := oldRaw.(*schema.Set)
		newSet := newRaw.(*schema.Set)

		for _, disablePrincipalRaw := range oldSet.Difference(newSet).List() {
			principal := disablePrincipalRaw.(string)
			input := &organizations.DisableAWSServiceAccessInput{
				ServicePrincipal: aws.String(principal),
			}

			log.Printf("[DEBUG] Disabling AWS Service Access in Organization: %s", input)
			_, err := conn.DisableAWSServiceAccess(input)

			if err != nil {
				return fmt.Errorf("error disabling AWS Service Access (%s) in Organization: %s", principal, err)
			}
		}

		for _, enablePrincipalRaw := range newSet.Difference(oldSet).List() {
			principal := enablePrincipalRaw.(string)
			input := &organizations.EnableAWSServiceAccessInput{
				ServicePrincipal: aws.String(principal),
			}

			log.Printf("[DEBUG] Enabling AWS Service Access in Organization: %s", input)
			_, err := conn.EnableAWSServiceAccess(input)

			if err != nil {
				return fmt.Errorf("error enabling AWS Service Access (%s) in Organization: %s", principal, err)
			}
		}
	}

	return resourceAwsOrganizationsOrganizationRead(d, meta)
}

func resourceAwsOrganizationsOrganizationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).organizationsconn

	log.Printf("[INFO] Deleting Organization: %s", d.Id())

	_, err := conn.DeleteOrganization(&organizations.DeleteOrganizationInput{})
	if err != nil {
		return fmt.Errorf("Error deleting Organization: %s", err)
	}

	return nil
}
