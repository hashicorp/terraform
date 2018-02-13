package aws

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsAcmCertificateValidation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAcmCertificateValidationCreate,
		Read:   resourceAwsAcmCertificateValidationRead,
		Delete: resourceAwsAcmCertificateValidationDelete,

		Schema: map[string]*schema.Schema{
			"certificate_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"validation_record_fqdns": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(45 * time.Minute),
		},
	}
}

func resourceAwsAcmCertificateValidationCreate(d *schema.ResourceData, meta interface{}) error {
	certificate_arn := d.Get("certificate_arn").(string)

	acmconn := meta.(*AWSClient).acmconn
	params := &acm.DescribeCertificateInput{
		CertificateArn: aws.String(certificate_arn),
	}

	resp, err := acmconn.DescribeCertificate(params)

	if err != nil {
		return fmt.Errorf("Error describing certificate: %s", err)
	}

	if *resp.Certificate.Type != "AMAZON_ISSUED" {
		return fmt.Errorf("Certificate %s has type %s, no validation necessary", *resp.Certificate.CertificateArn, *resp.Certificate.Type)
	}

	if validation_record_fqdns, ok := d.GetOk("validation_record_fqdns"); ok {
		err := resourceAwsAcmCertificateCheckValidationRecords(validation_record_fqdns.(*schema.Set).List(), resp.Certificate)
		if err != nil {
			return err
		}
	} else {
		log.Printf("[INFO] No validation_record_fqdns set, skipping check")
	}

	return resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		resp, err := acmconn.DescribeCertificate(params)

		if err != nil {
			return resource.NonRetryableError(fmt.Errorf("Error describing certificate: %s", err))
		}

		if *resp.Certificate.Status != "ISSUED" {
			return resource.RetryableError(fmt.Errorf("Expected certificate to be issued but was in state %s", *resp.Certificate.Status))
		}

		log.Printf("[INFO] ACM Certificate validation for %s done, certificate was issued", certificate_arn)
		return resource.NonRetryableError(resourceAwsAcmCertificateValidationRead(d, meta))
	})
}

func resourceAwsAcmCertificateCheckValidationRecords(validation_record_fqdns []interface{}, cert *acm.CertificateDetail) error {
	expected_fqdns := make([]string, len(cert.DomainValidationOptions))
	for i, v := range cert.DomainValidationOptions {
		if *v.ValidationMethod == acm.ValidationMethodDns {
			expected_fqdns[i] = strings.TrimSuffix(*v.ResourceRecord.Name, ".")
		}
	}

	actual_validation_record_fqdns := make([]string, 0, len(validation_record_fqdns))

	for _, v := range validation_record_fqdns {
		val := v.(string)
		actual_validation_record_fqdns = append(actual_validation_record_fqdns, strings.TrimSuffix(val, "."))
	}

	sort.Strings(expected_fqdns)
	sort.Strings(actual_validation_record_fqdns)

	log.Printf("[DEBUG] Checking validation_record_fqdns. Expected: %v, Actual: %v", expected_fqdns, actual_validation_record_fqdns)

	if !reflect.DeepEqual(expected_fqdns, actual_validation_record_fqdns) {
		return fmt.Errorf("Certificate needs %v to be set but only %v was passed to validation_record_fqdns", expected_fqdns, actual_validation_record_fqdns)
	}

	return nil
}

func resourceAwsAcmCertificateValidationRead(d *schema.ResourceData, meta interface{}) error {
	acmconn := meta.(*AWSClient).acmconn

	params := &acm.DescribeCertificateInput{
		CertificateArn: aws.String(d.Get("certificate_arn").(string)),
	}

	resp, err := acmconn.DescribeCertificate(params)

	if err != nil && isAWSErr(err, acm.ErrCodeResourceNotFoundException, "") {
		d.SetId("")
		return nil
	} else if err != nil {
		return fmt.Errorf("Error describing certificate: %s", err)
	}

	if *resp.Certificate.Status != "ISSUED" {
		log.Printf("[INFO] Certificate status not issued, was %s, tainting validation", *resp.Certificate.Status)
		d.SetId("")
	} else {
		d.SetId((*resp.Certificate.IssuedAt).String())
	}
	return nil
}

func resourceAwsAcmCertificateValidationDelete(d *schema.ResourceData, meta interface{}) error {
	// No need to do anything, certificate will be deleted when acm_certificate is deleted
	d.SetId("")
	return nil
}
