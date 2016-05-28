package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iot"
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
	conn := meta.(*AWSClient).iotconn

	log.Printf("[DEBUG] Creating certificate from csr")
	out, err := conn.CreateCertificateFromCsr(&iot.CreateCertificateFromCsrInput{
		CertificateSigningRequest: aws.String(d.Get("csr").(string)),
		SetAsActive:               aws.Bool(d.Get("active").(bool)),
	})

	if err != nil {
		log.Printf("[ERROR] %s", err)
		return err
	}
	log.Printf("[DEBUG] Created certificate from csr")

	d.SetId(*out.CertificateId)
	d.Set("arn", *out.CertificateArn)

	return nil
}

func resourceAwsIotCertificateRead(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AWSClient).iotconn

	out, err := conn.DescribeCertificate(&iot.DescribeCertificateInput{
		CertificateId: aws.String(d.Id()),
	})

	if err != nil {
		log.Printf("[ERROR] %s", err)
		return err
	}

	d.SetId(*out.CertificateDescription.CertificateId)
	d.Set("arn", *out.CertificateDescription.CertificateArn)

	return nil
}

func resourceAwsIotCertificateUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	if d.HasChange("csr") {
		//Make old cert inactive
		_, err := conn.UpdateCertificate(&iot.UpdateCertificateInput{
			CertificateId: aws.String(d.Id()),
			NewStatus:     aws.String("INACTIVE"),
		})

		if err != nil {
			log.Printf("[ERROR] %s", err)
			return nil
		}

		//TODO: Remove old cert??
		//conn.DeleteCertificate(&iot.DeleteCertificateInput{
		//	CertificateId: aws.String(d.Id()),
		//})

		out, err := conn.CreateCertificateFromCsr(&iot.CreateCertificateFromCsrInput{
			CertificateSigningRequest: aws.String(d.Get("csr").(string)),
			SetAsActive:               aws.Bool(d.Get("active")),
		})

		if err != nil {
			log.Printf("[ERROR] %s", err)
			return err
		}

		d.SetId(*out.CertificateId)
		d.Set("arn", *out.CertificateArn)

	} else {

		// What about a status like REVOKED etc. ??
		status := "INACTIVE"
		if d.Get("active").(bool) {
			status = "ACTIVE"
		}

		_, err := conn.UpdateCertificate(&iot.UpdateCertificateInput{
			CertificateId: aws.String(d.Id()),
			NewStatus:     aws.String(status),
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func resourceAwsIotCertificateDelete(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AWSClient).iotconn

	_, err := conn.UpdateCertificate(&iot.UpdateCertificateInput{
		CertificateId: aws.String(d.Id()),
		NewStatus:     aws.String("INACTIVE"),
	})

	if err != nil {
		log.Printf("[ERROR], %s", err)
		return err
	}

	_, err = conn.DeleteCertificate(&iot.DeleteCertificateInput{
		CertificateId: aws.String(d.Id()),
	})

	if err != nil {
		log.Printf("[ERROR] %s", err)
		return err
	}

	return nil
}
