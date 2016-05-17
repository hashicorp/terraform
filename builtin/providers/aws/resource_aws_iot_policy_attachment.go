package aws

import (
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
			},
		},
	}
}

func resourceAwsIotPolicyAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsIotPolicyAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsIotPolicyAttachmentUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsIotPolicyAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
