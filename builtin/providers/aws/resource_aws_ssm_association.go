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
			"association_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"instance_id": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"parameters": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"targets": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"values": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func resourceAwsSsmAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[DEBUG] SSM association create: %s", d.Id())

	assosciationInput := &ssm.CreateAssociationInput{
		Name: aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("instance_id"); ok {
		assosciationInput.InstanceId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("parameters"); ok {
		assosciationInput.Parameters = expandSSMDocumentParameters(v.(map[string]interface{}))
	}

	if _, ok := d.GetOk("targets"); ok {
		assosciationInput.Targets = expandAwsSsmTargets(d)
	}

	resp, err := ssmconn.CreateAssociation(assosciationInput)
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error creating SSM association: {{err}}", err)
	}

	if resp.AssociationDescription == nil {
		return fmt.Errorf("[ERROR] AssociationDescription was nil")
	}

	d.SetId(*resp.AssociationDescription.Name)
	d.Set("association_id", resp.AssociationDescription.AssociationId)

	return resourceAwsSsmAssociationRead(d, meta)
}

func resourceAwsSsmAssociationRead(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[DEBUG] Reading SSM Association: %s", d.Id())

	params := &ssm.DescribeAssociationInput{
		AssociationId: aws.String(d.Get("association_id").(string)),
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
	d.Set("association_id", association.AssociationId)

	if err := d.Set("targets", flattenAwsSsmTargets(association.Targets)); err != nil {
		return fmt.Errorf("[DEBUG] Error setting targets error: %#v", err)
	}

	return nil
}

func resourceAwsSsmAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[DEBUG] Deleting SSM Assosciation: %s", d.Id())

	params := &ssm.DeleteAssociationInput{
		AssociationId: aws.String(d.Get("association_id").(string)),
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
		values := make([]*string, 1)
		values[0] = aws.String(v.(string))
		docParams[k] = values
	}

	return docParams
}
