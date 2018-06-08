package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIotPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIotPolicyCreate,
		Read:   resourceAwsIotPolicyRead,
		Update: resourceAwsIotPolicyUpdate,
		Delete: resourceAwsIotPolicyDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"policy": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"default_version_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsIotPolicyCreate(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AWSClient).iotconn

	out, err := conn.CreatePolicy(&iot.CreatePolicyInput{
		PolicyName:     aws.String(d.Get("name").(string)),
		PolicyDocument: aws.String(d.Get("policy").(string)),
	})

	if err != nil {
		log.Printf("[ERROR] %s", err)
		return err
	}

	d.SetId(*out.PolicyName)

	return resourceAwsIotPolicyRead(d, meta)
}

func resourceAwsIotPolicyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	out, err := conn.GetPolicy(&iot.GetPolicyInput{
		PolicyName: aws.String(d.Id()),
	})

	if err != nil {
		log.Printf("[ERROR] %s", err)
		return err
	}

	d.Set("arn", out.PolicyArn)
	d.Set("default_version_id", out.DefaultVersionId)

	return nil
}

func resourceAwsIotPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	if d.HasChange("policy") {
		_, err := conn.CreatePolicyVersion(&iot.CreatePolicyVersionInput{
			PolicyName:     aws.String(d.Id()),
			PolicyDocument: aws.String(d.Get("policy").(string)),
			SetAsDefault:   aws.Bool(true),
		})

		if err != nil {
			log.Printf("[ERROR] %s", err)
			return err
		}
	}

	return resourceAwsIotPolicyRead(d, meta)
}

func resourceAwsIotPolicyDelete(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AWSClient).iotconn

	out, err := conn.ListPolicyVersions(&iot.ListPolicyVersionsInput{
		PolicyName: aws.String(d.Id()),
	})

	if err != nil {
		return err
	}

	// Delete all non-default versions of the policy
	for _, ver := range out.PolicyVersions {
		if !*ver.IsDefaultVersion {
			_, err = conn.DeletePolicyVersion(&iot.DeletePolicyVersionInput{
				PolicyName:      aws.String(d.Id()),
				PolicyVersionId: ver.VersionId,
			})
			if err != nil {
				log.Printf("[ERROR] %s", err)
				return err
			}
		}
	}

	//Delete default policy version
	_, err = conn.DeletePolicy(&iot.DeletePolicyInput{
		PolicyName: aws.String(d.Id()),
	})

	if err != nil {
		log.Printf("[ERROR] %s", err)
		return err
	}

	return nil
}
