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
	conn := meta.(*AWSClient).iotconn

	if d.HasChange("policies") {
		err := updatePolicies(conn, d)
		if err != nil {
			log.Printf("[ERROR] %v", err)
			return err
		}
	}

	return resourceAwsIotPolicyAttachmentRead(d, meta)
}

func updatePolicies(conn *iot.IoT, d *schema.ResourceData) error {
	o, n := d.GetChange("policies")
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
