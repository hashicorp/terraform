package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsAthenaNamedQuery() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAthenaNamedQueryCreate,
		Read:   resourceAwsAthenaNamedQueryRead,
		Delete: resourceAwsAthenaNamedQueryDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"query": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"database": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
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

	resp, err := conn.GetNamedQuery(input)
	if err != nil {
		if isAWSErr(err, athena.ErrCodeInvalidRequestException, d.Id()) {
			log.Printf("[WARN] Athena Named Query (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", resp.NamedQuery.Name)
	d.Set("query", resp.NamedQuery.QueryString)
	d.Set("database", resp.NamedQuery.Database)
	d.Set("description", resp.NamedQuery.Description)
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

	return nil
}
