package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codecommit"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsCodeCommitRepository() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCodeCommitRepositoryCreate,
		Update: resourceAwsCodeCommitRepositoryUpdate,
		Read:   resourceAwsCodeCommitRepositoryRead,
		Delete: resourceAwsCodeCommitRepositoryDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"repository_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) > 100 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 100 characters", k))
					}
					return
				},
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) > 1000 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 1000 characters", k))
					}
					return
				},
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"repository_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"clone_url_http": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"clone_url_ssh": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"default_branch": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAwsCodeCommitRepositoryCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codecommitconn

	input := &codecommit.CreateRepositoryInput{
		RepositoryName:        aws.String(d.Get("repository_name").(string)),
		RepositoryDescription: aws.String(d.Get("description").(string)),
	}

	out, err := conn.CreateRepository(input)
	if err != nil {
		return fmt.Errorf("Error creating CodeCommit Repository: %s", err)
	}

	d.SetId(d.Get("repository_name").(string))
	d.Set("repository_id", out.RepositoryMetadata.RepositoryId)
	d.Set("arn", out.RepositoryMetadata.Arn)
	d.Set("clone_url_http", out.RepositoryMetadata.CloneUrlHttp)
	d.Set("clone_url_ssh", out.RepositoryMetadata.CloneUrlSsh)

	return resourceAwsCodeCommitRepositoryUpdate(d, meta)
}

func resourceAwsCodeCommitRepositoryUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codecommitconn

	if _, ok := d.GetOk("default_branch"); ok {
		if d.HasChange("default_branch") {
			if err := resourceAwsCodeCommitUpdateDefaultBranch(conn, d); err != nil {
				return err
			}
		}
	}

	if d.HasChange("description") {
		if err := resourceAwsCodeCommitUpdateDescription(conn, d); err != nil {
			return err
		}
	}

	return resourceAwsCodeCommitRepositoryRead(d, meta)
}

func resourceAwsCodeCommitRepositoryRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codecommitconn

	input := &codecommit.GetRepositoryInput{
		RepositoryName: aws.String(d.Id()),
	}

	out, err := conn.GetRepository(input)
	if err != nil {
		return fmt.Errorf("Error reading CodeCommit Repository: %s", err.Error())
	}

	d.Set("repository_id", out.RepositoryMetadata.RepositoryId)
	d.Set("arn", out.RepositoryMetadata.Arn)
	d.Set("clone_url_http", out.RepositoryMetadata.CloneUrlHttp)
	d.Set("clone_url_ssh", out.RepositoryMetadata.CloneUrlSsh)
	d.Set("description", out.RepositoryMetadata.RepositoryDescription)
	d.Set("repository_name", out.RepositoryMetadata.RepositoryName)

	if _, ok := d.GetOk("default_branch"); ok {
		if out.RepositoryMetadata.DefaultBranch != nil {
			d.Set("default_branch", out.RepositoryMetadata.DefaultBranch)
		}
	}

	return nil
}

func resourceAwsCodeCommitRepositoryDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codecommitconn

	log.Printf("[DEBUG] CodeCommit Delete Repository: %s", d.Id())
	_, err := conn.DeleteRepository(&codecommit.DeleteRepositoryInput{
		RepositoryName: aws.String(d.Id()),
	})
	if err != nil {
		return fmt.Errorf("Error deleting CodeCommit Repository: %s", err.Error())
	}

	return nil
}

func resourceAwsCodeCommitUpdateDescription(conn *codecommit.CodeCommit, d *schema.ResourceData) error {
	branchInput := &codecommit.UpdateRepositoryDescriptionInput{
		RepositoryName:        aws.String(d.Id()),
		RepositoryDescription: aws.String(d.Get("description").(string)),
	}

	_, err := conn.UpdateRepositoryDescription(branchInput)
	if err != nil {
		return fmt.Errorf("Error Updating Repository Description for CodeCommit Repository: %s", err.Error())
	}

	return nil
}

func resourceAwsCodeCommitUpdateDefaultBranch(conn *codecommit.CodeCommit, d *schema.ResourceData) error {
	input := &codecommit.ListBranchesInput{
		RepositoryName: aws.String(d.Id()),
	}

	out, err := conn.ListBranches(input)
	if err != nil {
		return fmt.Errorf("Error reading CodeCommit Repository branches: %s", err.Error())
	}

	if len(out.Branches) == 0 {
		log.Printf("[WARN] Not setting Default Branch CodeCommit Repository that has no branches: %s", d.Id())
		return nil
	}

	branchInput := &codecommit.UpdateDefaultBranchInput{
		RepositoryName:    aws.String(d.Id()),
		DefaultBranchName: aws.String(d.Get("default_branch").(string)),
	}

	_, err = conn.UpdateDefaultBranch(branchInput)
	if err != nil {
		return fmt.Errorf("Error Updating Default Branch for CodeCommit Repository: %s", err.Error())
	}

	return nil
}
