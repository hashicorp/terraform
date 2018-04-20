package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/emr"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEMRSecurityConfiguration() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEmrSecurityConfigurationCreate,
		Read:   resourceAwsEmrSecurityConfigurationRead,
		Delete: resourceAwsEmrSecurityConfigurationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validateMaxLength(10280),
			},
			"name_prefix": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateMaxLength(10280 - resource.UniqueIDSuffixLength),
			},

			"configuration": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateJsonString,
			},

			"creation_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsEmrSecurityConfigurationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).emrconn

	var emrSCName string
	if v, ok := d.GetOk("name"); ok {
		emrSCName = v.(string)
	} else {
		if v, ok := d.GetOk("name_prefix"); ok {
			emrSCName = resource.PrefixedUniqueId(v.(string))
		} else {
			emrSCName = resource.PrefixedUniqueId("tf-emr-sc-")
		}
	}

	resp, err := conn.CreateSecurityConfiguration(&emr.CreateSecurityConfigurationInput{
		Name: aws.String(emrSCName),
		SecurityConfiguration: aws.String(d.Get("configuration").(string)),
	})

	if err != nil {
		return err
	}

	d.SetId(*resp.Name)
	return resourceAwsEmrSecurityConfigurationRead(d, meta)
}

func resourceAwsEmrSecurityConfigurationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).emrconn

	resp, err := conn.DescribeSecurityConfiguration(&emr.DescribeSecurityConfigurationInput{
		Name: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, "InvalidRequestException", "does not exist") {
			log.Printf("[WARN] EMR Security Configuraiton (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("creation_date", resp.CreationDateTime)
	d.Set("name", resp.Name)
	d.Set("configuration", resp.SecurityConfiguration)

	return nil
}

func resourceAwsEmrSecurityConfigurationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).emrconn

	_, err := conn.DeleteSecurityConfiguration(&emr.DeleteSecurityConfigurationInput{
		Name: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, "InvalidRequestException", "does not exist") {
			d.SetId("")
			return nil
		}
		return err
	}
	d.SetId("")

	return nil
}
