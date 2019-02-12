package aws

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsAcmCertificate() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAcmCertificateCreate,
		Read:   resourceAwsAcmCertificateRead,
		Update: resourceAwsAcmCertificateUpdate,
		Delete: resourceAwsAcmCertificateDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"certificate_body": {
				Type:      schema.TypeString,
				Optional:  true,
				StateFunc: normalizeCert,
			},

			"certificate_chain": {
				Type:      schema.TypeString,
				Optional:  true,
				StateFunc: normalizeCert,
			},
			"private_key": {
				Type:      schema.TypeString,
				Optional:  true,
				StateFunc: normalizeCert,
				Sensitive: true,
			},
			"domain_name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"private_key", "certificate_body", "certificate_chain"},
				StateFunc: func(v interface{}) string {
					// AWS Provider 1.42.0+ aws_route53_zone references may contain a
					// trailing period, which generates an ACM API error
					return strings.TrimSuffix(v.(string), ".")
				},
			},
			"subject_alternative_names": {
				Type:          schema.TypeList,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"private_key", "certificate_body", "certificate_chain"},
				Elem: &schema.Schema{
					Type: schema.TypeString,
					StateFunc: func(v interface{}) string {
						// AWS Provider 1.42.0+ aws_route53_zone references may contain a
						// trailing period, which generates an ACM API error
						return strings.TrimSuffix(v.(string), ".")
					},
				},
			},
			"validation_method": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"private_key", "certificate_body", "certificate_chain"},
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"domain_validation_options": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"domain_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"resource_record_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"resource_record_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"resource_record_value": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"validation_emails": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsAcmCertificateCreate(d *schema.ResourceData, meta interface{}) error {
	if _, ok := d.GetOk("domain_name"); ok {
		if _, ok := d.GetOk("validation_method"); !ok {
			return errors.New("validation_method must be set when creating a certificate")
		}
		return resourceAwsAcmCertificateCreateRequested(d, meta)
	} else if _, ok := d.GetOk("private_key"); ok {
		if _, ok := d.GetOk("certificate_body"); !ok {
			return errors.New("certificate_body must be set when importing a certificate with private_key")
		}
		return resourceAwsAcmCertificateCreateImported(d, meta)
	}
	return errors.New("certificate must be imported (private_key) or created (domain_name)")
}

func resourceAwsAcmCertificateCreateImported(d *schema.ResourceData, meta interface{}) error {
	acmconn := meta.(*AWSClient).acmconn
	resp, err := resourceAwsAcmCertificateImport(acmconn, d, false)
	if err != nil {
		return fmt.Errorf("Error importing certificate: %s", err)
	}

	d.SetId(*resp.CertificateArn)
	if v, ok := d.GetOk("tags"); ok {
		params := &acm.AddTagsToCertificateInput{
			CertificateArn: resp.CertificateArn,
			Tags:           tagsFromMapACM(v.(map[string]interface{})),
		}
		_, err := acmconn.AddTagsToCertificate(params)

		if err != nil {
			return fmt.Errorf("Error requesting certificate: %s", err)
		}
	}

	return resourceAwsAcmCertificateRead(d, meta)
}

func resourceAwsAcmCertificateCreateRequested(d *schema.ResourceData, meta interface{}) error {
	acmconn := meta.(*AWSClient).acmconn
	params := &acm.RequestCertificateInput{
		DomainName:       aws.String(strings.TrimSuffix(d.Get("domain_name").(string), ".")),
		ValidationMethod: aws.String(d.Get("validation_method").(string)),
	}

	if sans, ok := d.GetOk("subject_alternative_names"); ok {
		subjectAlternativeNames := make([]*string, len(sans.([]interface{})))
		for i, sanRaw := range sans.([]interface{}) {
			subjectAlternativeNames[i] = aws.String(strings.TrimSuffix(sanRaw.(string), "."))
		}
		params.SubjectAlternativeNames = subjectAlternativeNames
	}

	log.Printf("[DEBUG] ACM Certificate Request: %#v", params)
	resp, err := acmconn.RequestCertificate(params)

	if err != nil {
		return fmt.Errorf("Error requesting certificate: %s", err)
	}

	d.SetId(*resp.CertificateArn)
	if v, ok := d.GetOk("tags"); ok {
		params := &acm.AddTagsToCertificateInput{
			CertificateArn: resp.CertificateArn,
			Tags:           tagsFromMapACM(v.(map[string]interface{})),
		}
		_, err := acmconn.AddTagsToCertificate(params)

		if err != nil {
			return fmt.Errorf("Error requesting certificate: %s", err)
		}
	}

	return resourceAwsAcmCertificateRead(d, meta)
}

func resourceAwsAcmCertificateRead(d *schema.ResourceData, meta interface{}) error {
	acmconn := meta.(*AWSClient).acmconn

	params := &acm.DescribeCertificateInput{
		CertificateArn: aws.String(d.Id()),
	}

	return resource.Retry(time.Duration(1)*time.Minute, func() *resource.RetryError {
		resp, err := acmconn.DescribeCertificate(params)

		if err != nil {
			if isAWSErr(err, acm.ErrCodeResourceNotFoundException, "") {
				d.SetId("")
				return nil
			}
			return resource.NonRetryableError(fmt.Errorf("Error describing certificate: %s", err))
		}

		d.Set("domain_name", resp.Certificate.DomainName)
		d.Set("arn", resp.Certificate.CertificateArn)

		if err := d.Set("subject_alternative_names", cleanUpSubjectAlternativeNames(resp.Certificate)); err != nil {
			return resource.NonRetryableError(err)
		}

		domainValidationOptions, emailValidationOptions, err := convertValidationOptions(resp.Certificate)

		if err != nil {
			return resource.RetryableError(err)
		}

		if err := d.Set("domain_validation_options", domainValidationOptions); err != nil {
			return resource.NonRetryableError(err)
		}
		if err := d.Set("validation_emails", emailValidationOptions); err != nil {
			return resource.NonRetryableError(err)
		}

		d.Set("validation_method", resourceAwsAcmCertificateGuessValidationMethod(domainValidationOptions, emailValidationOptions))

		params := &acm.ListTagsForCertificateInput{
			CertificateArn: aws.String(d.Id()),
		}

		tagResp, err := acmconn.ListTagsForCertificate(params)
		if err != nil {
			return resource.NonRetryableError(fmt.Errorf("error listing tags for certificate (%s): %s", d.Id(), err))
		}
		if err := d.Set("tags", tagsToMapACM(tagResp.Tags)); err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})
}
func resourceAwsAcmCertificateGuessValidationMethod(domainValidationOptions []map[string]interface{}, emailValidationOptions []string) string {
	// The DescribeCertificate Response doesn't have information on what validation method was used
	// so we need to guess from the validation options we see...
	if len(domainValidationOptions) > 0 {
		return acm.ValidationMethodDns
	} else if len(emailValidationOptions) > 0 {
		return acm.ValidationMethodEmail
	} else {
		return "NONE"
	}
}

func resourceAwsAcmCertificateUpdate(d *schema.ResourceData, meta interface{}) error {
	acmconn := meta.(*AWSClient).acmconn

	if d.HasChange("private_key") || d.HasChange("certificate_body") || d.HasChange("certificate_chain") {
		_, err := resourceAwsAcmCertificateImport(acmconn, d, true)
		if err != nil {
			return fmt.Errorf("Error updating certificate: %s", err)
		}
	}

	if d.HasChange("tags") {
		err := setTagsACM(acmconn, d)
		if err != nil {
			return err
		}
	}
	return resourceAwsAcmCertificateRead(d, meta)
}

func cleanUpSubjectAlternativeNames(cert *acm.CertificateDetail) []string {
	sans := cert.SubjectAlternativeNames
	vs := make([]string, 0)
	for _, v := range sans {
		if aws.StringValue(v) != aws.StringValue(cert.DomainName) {
			vs = append(vs, aws.StringValue(v))
		}
	}
	return vs

}

func convertValidationOptions(certificate *acm.CertificateDetail) ([]map[string]interface{}, []string, error) {
	var domainValidationResult []map[string]interface{}
	var emailValidationResult []string

	if *certificate.Type == acm.CertificateTypeAmazonIssued {
		for _, o := range certificate.DomainValidationOptions {
			if o.ResourceRecord != nil {
				validationOption := map[string]interface{}{
					"domain_name":           *o.DomainName,
					"resource_record_name":  *o.ResourceRecord.Name,
					"resource_record_type":  *o.ResourceRecord.Type,
					"resource_record_value": *o.ResourceRecord.Value,
				}
				domainValidationResult = append(domainValidationResult, validationOption)
			} else if o.ValidationEmails != nil && len(o.ValidationEmails) > 0 {
				for _, validationEmail := range o.ValidationEmails {
					emailValidationResult = append(emailValidationResult, *validationEmail)
				}
			} else if o.ValidationStatus == nil || aws.StringValue(o.ValidationStatus) == acm.DomainStatusPendingValidation {
				log.Printf("[DEBUG] No validation options need to retry: %#v", o)
				return nil, nil, fmt.Errorf("No validation options need to retry: %#v", o)
			}
		}
	}

	return domainValidationResult, emailValidationResult, nil
}

func resourceAwsAcmCertificateDelete(d *schema.ResourceData, meta interface{}) error {
	acmconn := meta.(*AWSClient).acmconn

	log.Printf("[INFO] Deleting ACM Certificate: %s", d.Id())

	params := &acm.DeleteCertificateInput{
		CertificateArn: aws.String(d.Id()),
	}

	err := resource.Retry(10*time.Minute, func() *resource.RetryError {
		_, err := acmconn.DeleteCertificate(params)
		if err != nil {
			if isAWSErr(err, acm.ErrCodeResourceInUseException, "") {
				log.Printf("[WARN] Conflict deleting certificate in use: %s, retrying", err.Error())
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	if err != nil && !isAWSErr(err, acm.ErrCodeResourceNotFoundException, "") {
		return fmt.Errorf("Error deleting certificate: %s", err)
	}

	return nil
}

func resourceAwsAcmCertificateImport(conn *acm.ACM, d *schema.ResourceData, update bool) (*acm.ImportCertificateOutput, error) {
	params := &acm.ImportCertificateInput{
		PrivateKey:  []byte(d.Get("private_key").(string)),
		Certificate: []byte(d.Get("certificate_body").(string)),
	}
	if chain, ok := d.GetOk("certificate_chain"); ok {
		params.CertificateChain = []byte(chain.(string))
	}
	if update {
		params.CertificateArn = aws.String(d.Get("arn").(string))
	}

	log.Printf("[DEBUG] ACM Certificate Import: %#v", params)
	return conn.ImportCertificate(params)
}
