package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsAthenaNamedQuery() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAthenaNamedQueryCreate,
		Read:   resourceAwsAthenaNamedQueryRead,
		Delete: resourceAwsAthenaNamedQueryDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"query": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"database": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsAthenaNamedQueryCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).athenaconn

	input := &athena.CreateNamedQueryInput{
		Database:    aws.String(d.Get("database").(string)),
		Name:        aws.String(d.Get("name").(string)),
		QueryString: aws.String(d.Get("query").(string)),
	}
	if raw, ok := d.GetOk("description"); ok {
		input.Description = aws.String(raw.(string))
	}

	resp, err := conn.CreateNamedQuery(input)
	if err != nil {
		return err
	}
	d.SetId(*resp.NamedQueryId)
	return resourceAwsAthenaNamedQueryRead(d, meta)
}

func resourceAwsAthenaNamedQueryRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).athenaconn

	input := &athena.GetNamedQueryInput{
		NamedQueryId: aws.String(d.Id()),
	}

	_, err := conn.GetNamedQuery(input)
	if err != nil {
		if isAWSErr(err, athena.ErrCodeInvalidRequestException, d.Id()) {
			d.SetId("")
			return nil
		}
		return err
	}
	return nil
}

func resourceAwsAthenaNamedQueryDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).athenaconn

	input := &athena.DeleteNamedQueryInput{
		NamedQueryId: aws.String(d.Id()),
	}

	_, err := conn.DeleteNamedQuery(input)
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}
