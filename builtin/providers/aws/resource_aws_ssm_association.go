package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSsmAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSsmAssociationCreate,
		Read:   resourceAwsSsmAssociationRead,
		Delete: resourceAwsSsmAssociationDelete,

		Schema: map[string]*schema.Schema{
			"instance_id": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"parameters": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsSsmAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[DEBUG] SSM association create: %s", d.Id())

	assosciationInput := &ssm.CreateAssociationInput{
		Name:       aws.String(d.Get("name").(string)),
		InstanceId: aws.String(d.Get("instance_id").(string)),
	}

	if v, ok := d.GetOk("parameters"); ok {
		assosciationInput.Parameters = expandSSMDocumentParameters(v.(map[string]interface{}))
	}

	resp, err := ssmconn.CreateAssociation(assosciationInput)
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error creating SSM association: {{err}}", err)
	}

	if resp.AssociationDescription == nil {
		return fmt.Errorf("[ERROR] AssociationDescription was nil")
	}

	d.SetId(*resp.AssociationDescription.Name)

	return resourceAwsSsmAssociationRead(d, meta)
}

func resourceAwsSsmAssociationRead(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[DEBUG] Reading SSM Assosciation: %s", d.Id())

	params := &ssm.DescribeAssociationInput{
		Name:       aws.String(d.Get("name").(string)),
		InstanceId: aws.String(d.Get("instance_id").(string)),
	}

	resp, err := ssmconn.DescribeAssociation(params)

	if err != nil {
		return errwrap.Wrapf("[ERROR] Error reading SSM association: {{err}}", err)
	}
	if resp.AssociationDescription == nil {
		return fmt.Errorf("[ERROR] AssociationDescription was nil")
	}

	association := resp.AssociationDescription
	d.Set("instance_id", association.InstanceId)
	d.Set("name", association.Name)
	d.Set("parameters", association.Parameters)

	return nil
}

func resourceAwsSsmAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[DEBUG] Deleting SSM Assosciation: %s", d.Id())

	params := &ssm.DeleteAssociationInput{
		Name:       aws.String(d.Get("name").(string)),
		InstanceId: aws.String(d.Get("instance_id").(string)),
	}

	_, err := ssmconn.DeleteAssociation(params)

	if err != nil {
		return errwrap.Wrapf("[ERROR] Error deleting SSM association: {{err}}", err)
	}

	return nil
}

func expandSSMDocumentParameters(params map[string]interface{}) map[string][]*string {
	var docParams = make(map[string][]*string)
	for k, v := range params {
		var values []*string
		values[0] = aws.String(v.(string))
		docParams[k] = values
	}

	return docParams
}
