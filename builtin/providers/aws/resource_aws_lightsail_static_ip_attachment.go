package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsLightsailStaticIpAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLightsailStaticIpAttachmentCreate,
		Read:   resourceAwsLightsailStaticIpAttachmentRead,
		Delete: resourceAwsLightsailStaticIpAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"static_ip_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"instance_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsLightsailStaticIpAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lightsailconn

	staticIpName := d.Get("static_ip_name").(string)
	log.Printf("[INFO] Attaching Lightsail Static IP: %q", staticIpName)
	out, err := conn.AttachStaticIp(&lightsail.AttachStaticIpInput{
		StaticIpName: aws.String(staticIpName),
		InstanceName: aws.String(d.Get("instance_name").(string)),
	})
	if err != nil {
		return err
	}
	log.Printf("[INFO] Lightsail Static IP attached: %s", *out)

	d.SetId(staticIpName)

	return resourceAwsLightsailStaticIpAttachmentRead(d, meta)
}

func resourceAwsLightsailStaticIpAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lightsailconn

	staticIpName := d.Get("static_ip_name").(string)
	log.Printf("[INFO] Reading Lightsail Static IP: %q", staticIpName)
	out, err := conn.GetStaticIp(&lightsail.GetStaticIpInput{
		StaticIpName: aws.String(staticIpName),
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
	if !*out.StaticIp.IsAttached {
		log.Printf("[WARN] Lightsail Static IP (%s) is not attached, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	log.Printf("[INFO] Received Lightsail Static IP: %s", *out)

	d.Set("instance_name", out.StaticIp.AttachedTo)

	return nil
}

func resourceAwsLightsailStaticIpAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lightsailconn

	name := d.Get("static_ip_name").(string)
	log.Printf("[INFO] Detaching Lightsail Static IP: %q", name)
	out, err := conn.DetachStaticIp(&lightsail.DetachStaticIpInput{
		StaticIpName: aws.String(name),
	})
	if err != nil {
		return err
	}
	log.Printf("[INFO] Detached Lightsail Static IP: %s", *out)
	return nil
}
