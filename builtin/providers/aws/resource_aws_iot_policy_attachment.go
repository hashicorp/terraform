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
			"principals": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"policy": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsIotPolicyAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	for _, p := range d.Get("principals").(*schema.Set).List() {
		_, err := conn.AttachPrincipalPolicy(&iot.AttachPrincipalPolicyInput{
			Principal:  aws.String(d.Get("policy").(string)),
			PolicyName: aws.String(p.(string)),
		})

		if err != nil {
			log.Printf("[ERROR] %s", err)
			return err
		}
	}

	d.SetId(d.Get("name").(string))
	d.Set("policy", d.Get("policy").(string))
	d.Set("principals", d.Get("principals").(*schema.Set).List())
	return nil
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

	d.SetId(d.Get("name").(string))
	d.Set("policy", d.Get("policy").(string))
	d.Set("principals", principals)

	return nil
}

func resourceAwsIotPolicyAttachmentUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	if d.HasChange("principals") {
		err := updatePrincipalsPolicy(conn, d)
		if err != nil {
			log.Printf("[ERROR] %v", err)
			return err
		}
	}

	return resourceAwsIotPolicyAttachmentRead(d, meta)
}

func updatePrincipalsPolicy(conn *iot.IoT, d *schema.ResourceData) error {
	o, n := d.GetChange("principals")
	if o == nil {
		o = new(schema.Set)
	}
	if n == nil {
		n = new(schema.Set)
	}
	os := o.(*schema.Set)
	ns := n.(*schema.Set)

	toBeDetached := expandStringList(os.Difference(ns).List())
	toBeAttached := expandStringList(ns.Difference(os).List())

	policyName := d.Get("policy").(string)
	for _, p := range toBeDetached {
		_, err := conn.DetachPrincipalPolicy(&iot.DetachPrincipalPolicyInput{
			PolicyName: aws.String(policyName),
			Principal:  aws.String(*p),
		})

		if err != nil {
			return err
		}
	}

	for _, p := range toBeAttached {
		_, err := conn.AttachPrincipalPolicy(&iot.AttachPrincipalPolicyInput{
			PolicyName: aws.String(policyName),
			Principal:  aws.String(*p),
		})
		if err != nil {
			return err
		}
	}

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
