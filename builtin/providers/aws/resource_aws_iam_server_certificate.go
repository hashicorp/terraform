package aws

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIAMServerCertificate() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIAMServerCertificateCreate,
		Read:   resourceAwsIAMServerCertificateRead,
		Delete: resourceAwsIAMServerCertificateDelete,

		Schema: map[string]*schema.Schema{
			"certificate_body": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: normalizeCert,
			},

			"certificate_chain": &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				ForceNew:  true,
				StateFunc: normalizeCert,
			},

			"path": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"private_key": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: normalizeCert,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsIAMServerCertificateCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	createOpts := &iam.UploadServerCertificateInput{
		CertificateBody:       aws.String(d.Get("certificate_body").(string)),
		PrivateKey:            aws.String(d.Get("private_key").(string)),
		ServerCertificateName: aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("certificate_chain"); ok {
		createOpts.CertificateChain = aws.String(v.(string))
	}

	if v, ok := d.GetOk("Path"); ok {
		createOpts.Path = aws.String(v.(string))
	}

	resp, err := conn.UploadServerCertificate(createOpts)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error uploading server certificate, error: %s: %s", awsErr.Code(), awsErr.Message())
		}
		return fmt.Errorf("[WARN] Error uploading server certificate, error: %s", err)
	}

	d.SetId(*resp.ServerCertificateMetadata.ServerCertificateID)

	return resourceAwsIAMServerCertificateRead(d, meta)
}

func resourceAwsIAMServerCertificateRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	resp, err := conn.GetServerCertificate(&iam.GetServerCertificateInput{
		ServerCertificateName: aws.String(d.Get("name").(string)),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error reading IAM Server Certificate: %s: %s", awsErr.Code(), awsErr.Message())
		}
		return fmt.Errorf("[WARN] Error reading IAM Server Certificate: %s", err)
	}

	// these values should always be present, and have a default if not set in
	// configuration, and so safe to reference with nil checks
	d.Set("certificate_body", normalizeCert(resp.ServerCertificate.CertificateBody))
	d.Set("certificate_chain", normalizeCert(resp.ServerCertificate.CertificateChain))
	d.Set("path", resp.ServerCertificate.ServerCertificateMetadata.Path)
	d.Set("arn", resp.ServerCertificate.ServerCertificateMetadata.ARN)

	return nil
}

func resourceAwsIAMServerCertificateDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	_, err := conn.DeleteServerCertificate(&iam.DeleteServerCertificateInput{
		ServerCertificateName: aws.String(d.Get("name").(string)),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error deleting server certificate: %s: %s", awsErr.Code(), awsErr.Message())
		}
		return err
	}

	d.SetId("")
	return nil
}

func normalizeCert(cert interface{}) string {
	if cert == nil {
		return ""
	}
	switch cert.(type) {
	case string:
		hash := sha1.Sum([]byte(strings.TrimSpace(cert.(string))))
		return hex.EncodeToString(hash[:])
	case *string:
		hash := sha1.Sum([]byte(strings.TrimSpace(*cert.(*string))))
		return hex.EncodeToString(hash[:])
	default:
		return ""
	}
}
