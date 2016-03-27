package aws

import (
	"log"

	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEcrRepository() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEcrRepositoryCreate,
		Read:   resourceAwsEcrRepositoryRead,
		Delete: resourceAwsEcrRepositoryDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"registry_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"repository_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsEcrRepositoryCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrconn

	input := ecr.CreateRepositoryInput{
		RepositoryName: aws.String(d.Get("name").(string)),
	}

	log.Printf("[DEBUG] Creating ECR resository: %s", input)
	out, err := conn.CreateRepository(&input)
	if err != nil {
		return err
	}

	repository := *out.Repository

	log.Printf("[DEBUG] ECR repository created: %q", *repository.RepositoryArn)

	d.SetId(*repository.RepositoryName)
	d.Set("arn", *repository.RepositoryArn)
	d.Set("registry_id", *repository.RegistryId)

	return resourceAwsEcrRepositoryRead(d, meta)
}

func resourceAwsEcrRepositoryRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrconn

	log.Printf("[DEBUG] Reading repository %s", d.Id())
	out, err := conn.DescribeRepositories(&ecr.DescribeRepositoriesInput{
		RegistryId:      aws.String(d.Get("registry_id").(string)),
		RepositoryNames: []*string{aws.String(d.Id())},
	})
	if err != nil {
		if ecrerr, ok := err.(awserr.Error); ok && ecrerr.Code() == "RepositoryNotFoundException" {
			d.SetId("")
			return nil
		}
		return err
	}

	repository := out.Repositories[0]

	log.Printf("[DEBUG] Received repository %s", out)

	d.SetId(*repository.RepositoryName)
	d.Set("arn", *repository.RepositoryArn)
	d.Set("registry_id", *repository.RegistryId)

	repositoryUrl := buildRepositoryUrl(repository, meta.(*AWSClient).region)
	log.Printf("[INFO] Setting the repository url to be %s", repositoryUrl)
	d.Set("repository_url", repositoryUrl)

	return nil
}

func buildRepositoryUrl(repo *ecr.Repository, region string) string {
	return fmt.Sprintf("https://%s.dkr.ecr.%s.amazonaws.com/%s", *repo.RegistryId, region, *repo.RepositoryName)
}

func resourceAwsEcrRepositoryDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrconn

	_, err := conn.DeleteRepository(&ecr.DeleteRepositoryInput{
		RepositoryName: aws.String(d.Id()),
		RegistryId:     aws.String(d.Get("registry_id").(string)),
		Force:          aws.Bool(true),
	})
	if err != nil {
		if ecrerr, ok := err.(awserr.Error); ok && ecrerr.Code() == "RepositoryNotFoundException" {
			d.SetId("")
			return nil
		}
		return err
	}

	log.Printf("[DEBUG] repository %q deleted.", d.Get("arn").(string))

	return nil
}
