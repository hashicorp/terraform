package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/simpledb"
)

func resourceAwsSimpleDBDomain() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSimpleDBDomainCreate,
		Read:   resourceAwsSimpleDBDomainRead,
		Delete: resourceAwsSimpleDBDomainDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsSimpleDBDomainCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).simpledbconn

	name := d.Get("name").(string)
	input := &simpledb.CreateDomainInput{
		DomainName: aws.String(name),
	}
	_, err := conn.CreateDomain(input)
	if err != nil {
		return fmt.Errorf("Create SimpleDB Domain failed: %s", err)
	}

	d.SetId(name)
	return nil
}

func resourceAwsSimpleDBDomainRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).simpledbconn

	input := &simpledb.DomainMetadataInput{
		DomainName: aws.String(d.Id()),
	}
	_, err := conn.DomainMetadata(input)
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == "NoSuchDomain" {
			log.Printf("[WARN] Removing SimpleDB domain %q because it's gone.", d.Id())
			d.SetId("")
			return nil
		}
	}
	if err != nil {
		return err
	}

	d.Set("name", d.Id())
	return nil
}

func resourceAwsSimpleDBDomainDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).simpledbconn

	input := &simpledb.DeleteDomainInput{
		DomainName: aws.String(d.Id()),
	}
	_, err := conn.DeleteDomain(input)
	if err != nil {
		return fmt.Errorf("Delete SimpleDB Domain failed: %s", err)
	}

	return nil
}
