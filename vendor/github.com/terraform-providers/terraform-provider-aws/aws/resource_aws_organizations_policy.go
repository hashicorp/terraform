package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsOrganizationsPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsOrganizationsPolicyCreate,
		Read:   resourceAwsOrganizationsPolicyRead,
		Update: resourceAwsOrganizationsPolicyUpdate,
		Delete: resourceAwsOrganizationsPolicyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"content": {
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: suppressEquivalentAwsPolicyDiffs,
				ValidateFunc:     validation.ValidateJsonString,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  organizations.PolicyTypeServiceControlPolicy,
				ValidateFunc: validation.StringInSlice([]string{
					organizations.PolicyTypeServiceControlPolicy,
				}, false),
			},
		},
	}
}

func resourceAwsOrganizationsPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).organizationsconn

	// Description is required:
	// InvalidParameter: 1 validation error(s) found.
	// - missing required field, CreatePolicyInput.Description.
	input := &organizations.CreatePolicyInput{
		Content:     aws.String(d.Get("content").(string)),
		Description: aws.String(d.Get("description").(string)),
		Name:        aws.String(d.Get("name").(string)),
		Type:        aws.String(d.Get("type").(string)),
	}

	log.Printf("[DEBUG] Creating Organizations Policy: %s", input)

	var err error
	var resp *organizations.CreatePolicyOutput
	err = resource.Retry(4*time.Minute, func() *resource.RetryError {
		resp, err = conn.CreatePolicy(input)

		if err != nil {
			if isAWSErr(err, organizations.ErrCodeFinalizingOrganizationException, "") {
				log.Printf("[DEBUG] Trying to create policy again: %q", err.Error())
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error creating Organizations Policy: %s", err)
	}

	d.SetId(*resp.Policy.PolicySummary.Id)

	return resourceAwsOrganizationsPolicyRead(d, meta)
}

func resourceAwsOrganizationsPolicyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).organizationsconn

	input := &organizations.DescribePolicyInput{
		PolicyId: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Reading Organizations Policy: %s", input)
	resp, err := conn.DescribePolicy(input)
	if err != nil {
		if isAWSErr(err, organizations.ErrCodePolicyNotFoundException, "") {
			log.Printf("[WARN] Policy does not exist, removing from state: %s", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if resp.Policy == nil || resp.Policy.PolicySummary == nil {
		log.Printf("[WARN] Policy does not exist, removing from state: %s", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("arn", resp.Policy.PolicySummary.Arn)
	d.Set("content", resp.Policy.Content)
	d.Set("description", resp.Policy.PolicySummary.Description)
	d.Set("name", resp.Policy.PolicySummary.Name)
	d.Set("type", resp.Policy.PolicySummary.Type)
	return nil
}

func resourceAwsOrganizationsPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).organizationsconn

	input := &organizations.UpdatePolicyInput{
		PolicyId: aws.String(d.Id()),
	}

	if d.HasChange("content") {
		input.Content = aws.String(d.Get("content").(string))
	}

	if d.HasChange("description") {
		input.Description = aws.String(d.Get("description").(string))
	}

	if d.HasChange("name") {
		input.Name = aws.String(d.Get("name").(string))
	}

	log.Printf("[DEBUG] Updating Organizations Policy: %s", input)
	_, err := conn.UpdatePolicy(input)
	if err != nil {
		return fmt.Errorf("error updating Organizations Policy: %s", err)
	}

	return resourceAwsOrganizationsPolicyRead(d, meta)
}

func resourceAwsOrganizationsPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).organizationsconn

	input := &organizations.DeletePolicyInput{
		PolicyId: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deletion Organizations Policy: %s", input)
	_, err := conn.DeletePolicy(input)
	if err != nil {
		if isAWSErr(err, organizations.ErrCodePolicyNotFoundException, "") {
			return nil
		}
		return err
	}
	return nil
}
