package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEcrLifecyclePolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEcrLifecyclePolicyCreate,
		Read:   resourceAwsEcrLifecyclePolicyRead,
		Delete: resourceAwsEcrLifecyclePolicyDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"repository": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"policy": &schema.Schema{
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateFunc:     validateJsonString,
				DiffSuppressFunc: suppressEquivalentJsonDiffs,
			},
			"registry_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsEcrLifecyclePolicyCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrconn

	input := &ecr.PutLifecyclePolicyInput{
		RepositoryName:      aws.String(d.Get("repository").(string)),
		LifecyclePolicyText: aws.String(d.Get("policy").(string)),
	}

	resp, err := conn.PutLifecyclePolicy(input)
	if err != nil {
		return err
	}
	d.SetId(*resp.RepositoryName)
	d.Set("registry_id", resp.RegistryId)
	return resourceAwsEcrLifecyclePolicyRead(d, meta)
}

func resourceAwsEcrLifecyclePolicyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrconn

	input := &ecr.GetLifecyclePolicyInput{
		RepositoryName: aws.String(d.Id()),
	}

	resp, err := conn.GetLifecyclePolicy(input)
	if err != nil {
		if isAWSErr(err, ecr.ErrCodeRepositoryNotFoundException, "") {
			d.SetId("")
			return nil
		}
		if isAWSErr(err, ecr.ErrCodeLifecyclePolicyNotFoundException, "") {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("repository", resp.RepositoryName)
	d.Set("registry_id", resp.RegistryId)
	d.Set("policy", resp.LifecyclePolicyText)

	return nil
}

func resourceAwsEcrLifecyclePolicyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrconn

	input := &ecr.DeleteLifecyclePolicyInput{
		RepositoryName: aws.String(d.Id()),
	}

	_, err := conn.DeleteLifecyclePolicy(input)
	if err != nil {
		if isAWSErr(err, ecr.ErrCodeRepositoryNotFoundException, "") {
			d.SetId("")
			return nil
		}
		if isAWSErr(err, ecr.ErrCodeLifecyclePolicyNotFoundException, "") {
			d.SetId("")
			return nil
		}
		return err
	}

	return nil
}
