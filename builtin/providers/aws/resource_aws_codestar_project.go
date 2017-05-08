package aws

import (
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codestar"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsCodeStarProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCodeStarProjectCreate,
		Read:   resourceAwsCodeStarProjectRead,
		Update: resourceAwsCodeStarProjectUpdate,
		Delete: resourceAwsCodeStarProjectDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					val := v.(string)

					if len(val) > 15 {
						es = append(es, fmt.Errorf("%q must not be longer than 15 characters", k))
					}
					if !regexp.MustCompile("^[a-z][a-z0-9-]+$").MatchString(val) {
						es = append(es, fmt.Errorf("%q must start with a letter, only contain alphanumeric characters and hyphens", k))
					}

					return
				},
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsCodeStarProjectCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codestarconn
	out, err := conn.CreateProject(&codestar.CreateProjectInput{
		Description: aws.String(d.Get("description").(string)),
		Id:          aws.String(d.Get("name").(string)),
		Name:        aws.String(d.Get("name").(string)),
	})
	if err != nil {
		return err
	}

	d.SetId(aws.StringValue(out.Id))
	return resourceAwsCodeStarProjectRead(d, meta)
}

func resourceAwsCodeStarProjectRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codestarconn
	out, err := conn.DescribeProject(&codestar.DescribeProjectInput{
		Id: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}
	d.Set("name", aws.StringValue(out.Name))
	d.Set("description", aws.StringValue(out.Description))
	d.Set("arn", aws.StringValue(out.Arn))
	return nil
}

func resourceAwsCodeStarProjectUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codestarconn
	_, err := conn.UpdateProject(&codestar.UpdateProjectInput{
		Description: aws.String(d.Get("description").(string)),
		Name:        aws.String(d.Get("name").(string)),
		Id:          aws.String(d.Id()),
	})
	if err != nil {
		return err
	}
	return resourceAwsCodeStarProjectRead(d, meta)
}

func resourceAwsCodeStarProjectDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codestarconn
	_, err := conn.DeleteProject(&codestar.DeleteProjectInput{
		DeleteStack: aws.Bool(true),
		Id:          aws.String(d.Id()),
	})
	return err
}
