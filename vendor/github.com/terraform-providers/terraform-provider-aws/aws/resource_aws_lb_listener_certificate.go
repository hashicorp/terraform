package aws

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsLbListenerCertificate() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLbListenerCertificateCreate,
		Read:   resourceAwsLbListenerCertificateRead,
		Delete: resourceAwsLbListenerCertificateDelete,

		Schema: map[string]*schema.Schema{
			"listener_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"certificate_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsLbListenerCertificateCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elbv2conn

	params := &elbv2.AddListenerCertificatesInput{
		ListenerArn: aws.String(d.Get("listener_arn").(string)),
		Certificates: []*elbv2.Certificate{
			&elbv2.Certificate{
				CertificateArn: aws.String(d.Get("certificate_arn").(string)),
			},
		},
	}

	log.Printf("[DEBUG] Adding certificate: %s of listener: %s", d.Get("certificate_arn").(string), d.Get("listener_arn").(string))
	resp, err := conn.AddListenerCertificates(params)
	if err != nil {
		return fmt.Errorf("Error creating LB Listener Certificate: %s", err)
	}

	if len(resp.Certificates) == 0 {
		return errors.New("Error creating LB Listener Certificate: no certificates returned in response")
	}

	d.SetId(d.Get("listener_arn").(string) + "_" + d.Get("certificate_arn").(string))

	return resourceAwsLbListenerCertificateRead(d, meta)
}

func resourceAwsLbListenerCertificateRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elbv2conn

	certificateArn := d.Get("certificate_arn").(string)
	listenerArn := d.Get("listener_arn").(string)

	log.Printf("[DEBUG] Reading certificate: %s of listener: %s", certificateArn, listenerArn)

	var certificate *elbv2.Certificate
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		var err error
		certificate, err = findAwsLbListenerCertificate(certificateArn, listenerArn, true, nil, conn)
		if err != nil {
			return resource.NonRetryableError(err)
		}

		if certificate == nil {
			err = fmt.Errorf("certificate not found: %s", certificateArn)
			if d.IsNewResource() {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}

		return nil
	})
	if err != nil {
		if certificate == nil {
			log.Printf("[WARN] %s - removing from state", err)
			d.SetId("")
			return nil
		}
		return err
	}

	return nil
}

func resourceAwsLbListenerCertificateDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elbv2conn
	log.Printf("[DEBUG] Deleting certificate: %s of listener: %s", d.Get("certificate_arn").(string), d.Get("listener_arn").(string))

	params := &elbv2.RemoveListenerCertificatesInput{
		ListenerArn: aws.String(d.Get("listener_arn").(string)),
		Certificates: []*elbv2.Certificate{
			&elbv2.Certificate{
				CertificateArn: aws.String(d.Get("certificate_arn").(string)),
			},
		},
	}

	_, err := conn.RemoveListenerCertificates(params)
	if err != nil {
		if isAWSErr(err, elbv2.ErrCodeCertificateNotFoundException, "") {
			return nil
		}
		if isAWSErr(err, elbv2.ErrCodeListenerNotFoundException, "") {
			return nil
		}
		return fmt.Errorf("Error removing LB Listener Certificate: %s", err)
	}

	return nil
}

func findAwsLbListenerCertificate(certificateArn, listenerArn string, skipDefault bool, nextMarker *string, conn *elbv2.ELBV2) (*elbv2.Certificate, error) {
	params := &elbv2.DescribeListenerCertificatesInput{
		ListenerArn: aws.String(listenerArn),
		PageSize:    aws.Int64(400),
	}
	if nextMarker != nil {
		params.Marker = nextMarker
	}

	resp, err := conn.DescribeListenerCertificates(params)
	if err != nil {
		return nil, err
	}

	for _, cert := range resp.Certificates {
		if skipDefault && *cert.IsDefault {
			continue
		}

		if *cert.CertificateArn == certificateArn {
			return cert, nil
		}
	}

	if resp.NextMarker != nil {
		return findAwsLbListenerCertificate(certificateArn, listenerArn, skipDefault, resp.NextMarker, conn)
	}
	return nil, nil
}
