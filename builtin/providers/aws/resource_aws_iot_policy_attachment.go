package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIotPolicyAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIotPolicyAttachmentCreate,
		Read:   resourceAwsIotPolicyAttachmentRead,
		Update: resourceAwsIotPolicyAttachmentUpdate,
		Delete: resourceAwsIotPolicyAttachmentDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"principal": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"policies": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceAwsIotPolicyAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	for _, p := range d.Get("policies").(*schema.Set).List() {
		_, err := conn.AttachPrincipalPolicy(&iot.AttachPrincipalPolicyInput{
			Principal:  aws.String(d.Get("principal").(string)),
			PolicyName: aws.String(p.(string)),
		})

		if err != nil {
			log.Printf("[ERROR] %s", err)
			return err
		}
	}

	d.SetId(d.Get("name").(string))
	d.Set("principal", d.Get("principal").(string))
	d.Set("policies", d.Get("policies").(*schema.Set).List())
	return nil
}

func resourceAwsIotPolicyAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	out, err := conn.ListPrincipalPolicies(&iot.ListPrincipalPoliciesInput{
		Principal: aws.String(d.Get("principal").(string)),
	})

	if err != nil {
		log.Printf("[ERROR] %s", err)
		return err
	}

	policies := make([]string, len(out.Policies))
	for i, p := range out.Policies {
		policies[i] = *p.PolicyName
	}

	d.SetId(d.Get("name").(string))
	d.Set("principal", d.Get("principal").(string))
	d.Set("policies", policies)

	return nil
}

func resourceAwsIotPolicyAttachmentUpdate(d *schema.ResourceData, meta interface{}) error {
	//TODO: implement
	return nil
}

func resourceAwsIotPolicyAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	for _, p := range d.Get("policies").(*schema.Set).List() {
		log.Printf("[INFO] %+v", p)
		_, err := conn.DetachPrincipalPolicy(&iot.DetachPrincipalPolicyInput{
			Principal:  aws.String(d.Get("principal").(string)),
			PolicyName: aws.String(p.(string)),
		})

		if err != nil {
			log.Printf("[ERROR] %s", err)
			return err
		}
	}
	return nil
}
