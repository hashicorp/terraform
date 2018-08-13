package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsOrganizationsPolicyAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsOrganizationsPolicyAttachmentCreate,
		Read:   resourceAwsOrganizationsPolicyAttachmentRead,
		Delete: resourceAwsOrganizationsPolicyAttachmentDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"policy_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"target_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsOrganizationsPolicyAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).organizationsconn

	policyID := d.Get("policy_id").(string)
	targetID := d.Get("target_id").(string)

	input := &organizations.AttachPolicyInput{
		PolicyId: aws.String(policyID),
		TargetId: aws.String(targetID),
	}

	log.Printf("[DEBUG] Creating Organizations Policy Attachment: %s", input)

	err := resource.Retry(4*time.Minute, func() *resource.RetryError {
		_, err := conn.AttachPolicy(input)

		if err != nil {
			if isAWSErr(err, organizations.ErrCodeFinalizingOrganizationException, "") {
				log.Printf("[DEBUG] Trying to create policy attachment again: %q", err.Error())
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error creating Organizations Policy Attachment: %s", err)
	}

	d.SetId(fmt.Sprintf("%s:%s", targetID, policyID))

	return resourceAwsOrganizationsPolicyAttachmentRead(d, meta)
}

func resourceAwsOrganizationsPolicyAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).organizationsconn

	targetID, policyID, err := decodeAwsOrganizationsPolicyAttachmentID(d.Id())
	if err != nil {
		return err
	}

	input := &organizations.ListPoliciesForTargetInput{
		Filter:   aws.String(organizations.PolicyTypeServiceControlPolicy),
		TargetId: aws.String(targetID),
	}

	log.Printf("[DEBUG] Listing Organizations Policies for Target: %s", input)
	var output *organizations.PolicySummary
	err = conn.ListPoliciesForTargetPages(input, func(page *organizations.ListPoliciesForTargetOutput, lastPage bool) bool {
		for _, policySummary := range page.Policies {
			if aws.StringValue(policySummary.Id) == policyID {
				output = policySummary
				return true
			}
		}
		return !lastPage
	})

	if err != nil {
		if isAWSErr(err, organizations.ErrCodeTargetNotFoundException, "") {
			log.Printf("[WARN] Target does not exist, removing from state: %s", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if output == nil {
		log.Printf("[WARN] Attachment does not exist, removing from state: %s", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("policy_id", policyID)
	d.Set("target_id", targetID)
	return nil
}

func resourceAwsOrganizationsPolicyAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).organizationsconn

	targetID, policyID, err := decodeAwsOrganizationsPolicyAttachmentID(d.Id())
	if err != nil {
		return err
	}

	input := &organizations.DetachPolicyInput{
		PolicyId: aws.String(policyID),
		TargetId: aws.String(targetID),
	}

	log.Printf("[DEBUG] Detaching Organizations Policy %q from %q", policyID, targetID)
	_, err = conn.DetachPolicy(input)
	if err != nil {
		if isAWSErr(err, organizations.ErrCodePolicyNotFoundException, "") {
			return nil
		}
		if isAWSErr(err, organizations.ErrCodeTargetNotFoundException, "") {
			return nil
		}
		return err
	}
	return nil
}

func decodeAwsOrganizationsPolicyAttachmentID(id string) (string, string, error) {
	idParts := strings.Split(id, ":")
	if len(idParts) != 2 {
		return "", "", fmt.Errorf("expected ID in format of TARGETID:POLICYID, received: %s", id)
	}
	return idParts[0], idParts[1], nil
}
