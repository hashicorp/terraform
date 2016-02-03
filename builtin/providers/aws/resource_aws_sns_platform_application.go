package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSnsPlatformApplicationAPNS() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSnsPlatformApplicationAPNSCreate,
		Read:   resourceAwsSnsPlatformApplicationAPNSRead,
		Update: resourceAwsSnsPlatformApplicationAPNSUpdate,
		Delete: resourceAwsSnsPlatformApplicationDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value != "SANDBOX" {
						errors = append(errors, fmt.Errorf(
							"%q has unsupported value %q", k, value))
					}
					return
				},
			},
			"platform_credential": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"platform_principal": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsSnsPlatformApplicationAPNSCreate(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	platform := "APNS"
	if d.Get("type") != nil {
		platform += "_" + d.Get("type").(string)
	}

	resp, err := snsconn.CreatePlatformApplication(&sns.CreatePlatformApplicationInput{
		Platform: aws.String(platform),
		Name:     aws.String(d.Get("name").(string)),
		Attributes: aws.StringMap(map[string]string{
			"PlatformCredential": d.Get("platform_credential").(string),
			"PlatformPrincipal":  d.Get("platform_principal").(string),
		}),
	})

	if err != nil {
		return fmt.Errorf("Error creating platform application %s", err)
	}

	d.SetId(*resp.PlatformApplicationArn)

	return resourceAwsSnsPlatformApplicationAPNSRead(d, meta)
}

func resourceAwsSnsPlatformApplicationAPNSRead(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	resp, err := snsconn.GetPlatformApplicationAttributes(&sns.GetPlatformApplicationAttributesInput{
		PlatformApplicationArn: aws.String(d.Id()),
	})

	if err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == "NotFound" {
			log.Printf("[DEBUG] Error reading platform application - not found : %s", resp)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading platform application %s", err)
	}

	return nil
}

func resourceAwsSnsPlatformApplicationAPNSUpdate(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	resp, err := snsconn.SetPlatformApplicationAttributes(&sns.SetPlatformApplicationAttributesInput{
		PlatformApplicationArn: aws.String(d.Id()),
		Attributes: aws.StringMap(map[string]string{
			"PlatformCredential": d.Get("platform_credential").(string),
			"PlatformPrincipal":  d.Get("platform_principal").(string),
		}),
	})

	if err != nil {
		return fmt.Errorf("Error reading platform application %s", err)
	}

	log.Printf("[DEBUG] Received GCM platform application: %s", resp)

	return nil
}

func resourceAwsSnsPlatformApplicationDelete(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	_, err := snsconn.DeletePlatformApplication(&sns.DeletePlatformApplicationInput{
		PlatformApplicationArn: aws.String(d.Id()),
	})

	if err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == "NotFound" {
			return nil
		}
		return fmt.Errorf("Error deleting platform application %s", err)
	}

	return nil
}

func resourceAwsSnsPlatformApplicationGCM() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSnsPlatformApplicationGCMCreate,
		Read:   resourceAwsSnsPlatformApplicationGCMRead,
		Update: resourceAwsSnsPlatformApplicationGCMUpdate,
		Delete: resourceAwsSnsPlatformApplicationDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"platform_credential": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsSnsPlatformApplicationGCMCreate(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	resp, err := snsconn.CreatePlatformApplication(&sns.CreatePlatformApplicationInput{
		Name:     aws.String(d.Get("name").(string)),
		Platform: aws.String("GCM"),
		Attributes: aws.StringMap(map[string]string{
			"PlatformCredential": d.Get("platform_credential").(string),
		}),
	})
	if err != nil {
		return fmt.Errorf("Error reading platform application %s", err)
	}

	d.SetId(*resp.PlatformApplicationArn)

	return resourceAwsSnsPlatformApplicationGCMRead(d, meta)
}

func resourceAwsSnsPlatformApplicationGCMRead(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	_, err := snsconn.GetPlatformApplicationAttributes(&sns.GetPlatformApplicationAttributesInput{
		PlatformApplicationArn: aws.String(d.Id()),
	})

	if err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == "NotFound" {
			log.Printf("[DEBUG] Error reading platform application - not found : %s", err)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading platform application %s", err)
	}

	return nil
}

func resourceAwsSnsPlatformApplicationGCMUpdate(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	_, err := snsconn.SetPlatformApplicationAttributes(&sns.SetPlatformApplicationAttributesInput{
		PlatformApplicationArn: aws.String(d.Id()),
		Attributes: aws.StringMap(map[string]string{
			"PlatformCredential": d.Get("platform_credential").(string),
		}),
	})

	if err != nil {
		return fmt.Errorf("Error updating platform application %s", err)
	}

	return nil
}
