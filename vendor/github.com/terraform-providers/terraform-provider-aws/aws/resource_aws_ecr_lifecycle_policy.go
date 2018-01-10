package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEcrLifecyclePolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEcrLifecyclePolicyCreate,
		Read:   resourceAwsEcrLifecyclePolicyRead,
		Delete: resourceAwsEcrLifecyclePolicyDelete,

		Schema: map[string]*schema.Schema{
			"repository": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"policy": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateJsonString,
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
		RegistryId:     aws.String(d.Get("registry_id").(string)),
		RepositoryName: aws.String(d.Get("repository").(string)),
	}

	_, err := conn.GetLifecyclePolicy(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeRepositoryNotFoundException, ecr.ErrCodeLifecyclePolicyNotFoundException:
				d.SetId("")
				return nil
			default:
				return err
			}
		}
		return err
	}

	return nil
}

func resourceAwsEcrLifecyclePolicyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrconn

	input := &ecr.DeleteLifecyclePolicyInput{
		RegistryId:     aws.String(d.Get("registry_id").(string)),
		RepositoryName: aws.String(d.Get("repository").(string)),
	}

	_, err := conn.DeleteLifecyclePolicy(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeRepositoryNotFoundException, ecr.ErrCodeLifecyclePolicyNotFoundException:
				d.SetId("")
				return nil
			default:
				return err
			}
		}
		return err
	}

	return nil
}
