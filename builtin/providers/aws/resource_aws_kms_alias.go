package aws

import (
	"fmt"
	"log"
	"regexp"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
)

func resourceAwsKmsAlias() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsKmsAliasCreate,
		Read:   resourceAwsKmsAliasRead,
		Update: resourceAwsKmsAliasUpdate,
		Delete: resourceAwsKmsAliasDelete,

		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(string)
					if !regexp.MustCompile(`^(alias\/)[a-zA-Z0-9:/_-]+$`).MatchString(value) {
						es = append(es, fmt.Errorf(
							"name must begin with 'alias/' and be comprised of only [a-zA-Z0-9:/_-]", k))
					}
					return
				},
			},
			"target_key_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
		},
	}
}

func resourceAwsKmsAliasCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn
	name := d.Get("name").(string)
	targetKeyId := d.Get("target_key_id").(string)

	log.Printf("[DEBUG] KMS alias create name: %s, target_key: %s", name, targetKeyId)

	req := &kms.CreateAliasInput{
		AliasName: 	 aws.String(name),
		TargetKeyId: aws.String(targetKeyId),
	}
	_, err := conn.CreateAlias(req)
	if err != nil {
		return err
	}
	d.SetId(name)
	return resourceAwsKmsAliasRead(d, meta)
}

func resourceAwsKmsAliasRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn
	name := d.Get("name").(string)

	req := &kms.ListAliasesInput{}
	resp, err := conn.ListAliases(req)
	if err != nil {
		return err
	}
	for _,e := range resp.Aliases {
		if name == *e.AliasName {
			if err := d.Set("arn", e.AliasArn); err != nil {
				return err
			}
			if err := d.Set("target_key_id", e.TargetKeyId); err != nil {
				return err
			}
			return nil
		}
	}

	log.Printf("[DEBUG] KMS alias read: alias not found")
	d.SetId("")
	return nil
}

func resourceAwsKmsAliasUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn

	if d.HasChange("target_key_id") {
		if err := resourceAwsKmsAliasTargetUpdate(conn, d); err != nil {
			return err
		}
	}
	return nil
}

func resourceAwsKmsAliasTargetUpdate(conn *kms.KMS, d *schema.ResourceData) error {
	name := d.Get("name").(string)
	targetKeyId := d.Get("target_key_id").(string)

	log.Printf("[DEBUG] KMS alias: %s, update target: %s", name, targetKeyId)

	req := &kms.UpdateAliasInput{
		AliasName: 	 aws.String(name),
		TargetKeyId: aws.String(targetKeyId),
	}
	_, err := conn.UpdateAlias(req)

	return err
}

func resourceAwsKmsAliasDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn
	name := d.Get("name").(string)

	req := &kms.DeleteAliasInput{
		AliasName: 	 aws.String(name),
	}
	_, err := conn.DeleteAlias(req)

	log.Printf("[DEBUG] KMS Alias: %s deleted.", name)
	d.SetId("")
	return err
}
