package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsLightsailStaticIp() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLightsailStaticIpCreate,
		Read:   resourceAwsLightsailStaticIpRead,
		Delete: resourceAwsLightsailStaticIpDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"support_code": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsLightsailStaticIpCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lightsailconn

	name := d.Get("name").(string)
	log.Printf("[INFO] Allocating Lightsail Static IP: %q", name)
	out, err := conn.AllocateStaticIp(&lightsail.AllocateStaticIpInput{
		StaticIpName: aws.String(name),
	})
	if err != nil {
		return err
	}
	log.Printf("[INFO] Lightsail Static IP allocated: %s", *out)

	d.SetId(name)

	return resourceAwsLightsailStaticIpRead(d, meta)
}

func resourceAwsLightsailStaticIpRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lightsailconn

	name := d.Get("name").(string)
	log.Printf("[INFO] Reading Lightsail Static IP: %q", name)
	out, err := conn.GetStaticIp(&lightsail.GetStaticIpInput{
		StaticIpName: aws.String(name),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NotFoundException" {
				log.Printf("[WARN] Lightsail Static IP (%s) not found, removing from state", d.Id())
				d.SetId("")
				return nil
			}
		}
		return err
	}
	log.Printf("[INFO] Received Lightsail Static IP: %s", *out)

	d.Set("arn", out.StaticIp.Arn)
	d.Set("ip_address", out.StaticIp.IpAddress)
	d.Set("support_code", out.StaticIp.SupportCode)

	return nil
}

func resourceAwsLightsailStaticIpDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lightsailconn

	name := d.Get("name").(string)
	log.Printf("[INFO] Deleting Lightsail Static IP: %q", name)
	out, err := conn.ReleaseStaticIp(&lightsail.ReleaseStaticIpInput{
		StaticIpName: aws.String(name),
	})
	if err != nil {
		return err
	}
	log.Printf("[INFO] Deleted Lightsail Static IP: %s", *out)
	return nil
}
