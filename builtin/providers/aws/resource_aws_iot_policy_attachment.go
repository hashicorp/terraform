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
		Delete: resourceAwsIotPolicyAttachmentDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"principals": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ForceNew: true,
			},
			"policy": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsIotPolicyAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	for _, p := range d.Get("principals").(*schema.Set).List() {
		_, err := conn.AttachPrincipalPolicy(&iot.AttachPrincipalPolicyInput{
			Principal:  aws.String(p.(string)),
			PolicyName: aws.String(d.Get("policy").(string)),
		})

		if err != nil {
			log.Printf("[ERROR] %s", err)
			return err
		}
	}

	d.SetId(d.Get("name").(string))

	return resourceAwsIotPolicyAttachmentRead(d, meta)
}

func resourceAwsIotPolicyAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	out, err := conn.ListPolicyPrincipals(&iot.ListPolicyPrincipalsInput{
		PolicyName: aws.String(d.Get("policy").(string)),
	})

	if err != nil {
		log.Printf("[ERROR] %s", err)
		return err
	}

	principals := make([]string, len(out.Principals))
	for i, p := range out.Principals {
		principals[i] = *p
	}

	d.Set("policy", d.Get("policy").(string))
	d.Set("principals", principals)

	return nil
}

func resourceAwsIotPolicyAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	for _, p := range d.Get("principals").(*schema.Set).List() {
		log.Printf("[INFO] %+v", p)
		_, err := conn.DetachPrincipalPolicy(&iot.DetachPrincipalPolicyInput{
			Principal:  aws.String(p.(string)),
			PolicyName: aws.String(d.Get("policy").(string)),
		})

		if err != nil {
			log.Printf("[ERROR] %s", err)
			return err
		}
	}
	return nil
}
