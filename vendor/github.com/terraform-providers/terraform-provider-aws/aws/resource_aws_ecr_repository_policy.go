package aws

import (
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEcrRepositoryPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEcrRepositoryPolicyCreate,
		Read:   resourceAwsEcrRepositoryPolicyRead,
		Update: resourceAwsEcrRepositoryPolicyUpdate,
		Delete: resourceAwsEcrRepositoryPolicyDelete,

		Schema: map[string]*schema.Schema{
			"repository": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"policy": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"registry_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsEcrRepositoryPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrconn

	input := ecr.SetRepositoryPolicyInput{
		RepositoryName: aws.String(d.Get("repository").(string)),
		PolicyText:     aws.String(d.Get("policy").(string)),
	}

	log.Printf("[DEBUG] Creating ECR resository policy: %s", input)

	// Retry due to IAM eventual consistency
	var out *ecr.SetRepositoryPolicyOutput
	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		var err error
		out, err = conn.SetRepositoryPolicy(&input)

		if isAWSErr(err, "InvalidParameterException", "Invalid repository policy provided") {
			return resource.RetryableError(err)

		}
		return resource.NonRetryableError(err)
	})
	if err != nil {
		return err
	}

	repositoryPolicy := *out

	log.Printf("[DEBUG] ECR repository policy created: %s", *repositoryPolicy.RepositoryName)

	d.SetId(*repositoryPolicy.RepositoryName)
	d.Set("registry_id", repositoryPolicy.RegistryId)

	return resourceAwsEcrRepositoryPolicyRead(d, meta)
}

func resourceAwsEcrRepositoryPolicyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrconn

	log.Printf("[DEBUG] Reading repository policy %s", d.Id())
	out, err := conn.GetRepositoryPolicy(&ecr.GetRepositoryPolicyInput{
		RegistryId:     aws.String(d.Get("registry_id").(string)),
		RepositoryName: aws.String(d.Id()),
	})
	if err != nil {
		if ecrerr, ok := err.(awserr.Error); ok {
			switch ecrerr.Code() {
			case "RepositoryNotFoundException", "RepositoryPolicyNotFoundException":
				d.SetId("")
				return nil
			default:
				return err
			}
		}
		return err
	}

	log.Printf("[DEBUG] Received repository policy %s", out)

	repositoryPolicy := out

	d.SetId(*repositoryPolicy.RepositoryName)
	d.Set("registry_id", repositoryPolicy.RegistryId)

	return nil
}

func resourceAwsEcrRepositoryPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrconn

	if !d.HasChange("policy") {
		return nil
	}

	input := ecr.SetRepositoryPolicyInput{
		RepositoryName: aws.String(d.Get("repository").(string)),
		RegistryId:     aws.String(d.Get("registry_id").(string)),
		PolicyText:     aws.String(d.Get("policy").(string)),
	}

	log.Printf("[DEBUG] Updating ECR resository policy: %s", input)

	// Retry due to IAM eventual consistency
	var out *ecr.SetRepositoryPolicyOutput
	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		var err error
		out, err = conn.SetRepositoryPolicy(&input)

		if isAWSErr(err, "InvalidParameterException", "Invalid repository policy provided") {
			return resource.RetryableError(err)

		}
		return resource.NonRetryableError(err)
	})
	if err != nil {
		return err
	}

	repositoryPolicy := *out

	d.SetId(*repositoryPolicy.RepositoryName)
	d.Set("registry_id", repositoryPolicy.RegistryId)

	return nil
}

func resourceAwsEcrRepositoryPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrconn

	_, err := conn.DeleteRepositoryPolicy(&ecr.DeleteRepositoryPolicyInput{
		RepositoryName: aws.String(d.Id()),
		RegistryId:     aws.String(d.Get("registry_id").(string)),
	})
	if err != nil {
		if ecrerr, ok := err.(awserr.Error); ok {
			switch ecrerr.Code() {
			case "RepositoryNotFoundException", "RepositoryPolicyNotFoundException":
				d.SetId("")
				return nil
			default:
				return err
			}
		}
		return err
	}

	log.Printf("[DEBUG] repository policy %s deleted.", d.Id())

	return nil
}
