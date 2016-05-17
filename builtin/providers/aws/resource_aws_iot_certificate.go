package aws

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIotCertificate() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIotCertificateCreate,
		Read:   resourceAwsIotCertificateRead,
		Update: resourceAwsIotCertificateUpdate,
		Delete: resourceAwsIotCertificateUpdate,
		Schema: map[string]*schema.Schema{
			"csr": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"active": &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
			},
		},
	}
}

func resourceAwsIotCertificateCreate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsIotCertificateRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsIotCertificateUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsIotCertificateDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
