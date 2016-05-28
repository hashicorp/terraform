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
			"principal": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"policies": &schema.Schema{
				Type:     schema.TypeList,
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

	for _, p := range d.Get("policies").([]string) {
		_, err := conn.AttachPrincipalPolicy(&iot.AttachPrincipalPolicyInput{
			Principal:  aws.String(d.Get("principal").(string)),
			PolicyName: aws.String(p),
		})

		if err != nil {
			log.Printf("[ERROR] %s", err)
			return err
		}
	}

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

	length := len(out.Policies)
	policies := make([]string, length)
	for i, p := range out.Policies {
		policies[i] = *p.PolicyName
	}

	d.Set("policies", policies)

	return nil
}

func resourceAwsIotPolicyAttachmentUpdate(d *schema.ResourceData, meta interface{}) error {
	//TODO: implement
	return nil
}

func resourceAwsIotPolicyAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	for _, p := range d.Get("policies").([]string) {
		_, err := conn.DetachPrincipalPolicy(&iot.DetachPrincipalPolicyInput{
			Principal:  aws.String(d.Get("principal").(string)),
			PolicyName: aws.String(p),
		})

		if err != nil {
			log.Printf("[ERROR] %s", err)
			return err
		}
	}
	return nil
}
