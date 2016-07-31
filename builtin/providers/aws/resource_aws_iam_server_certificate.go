package aws

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
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
				Type:     schema.TypeString,
				Optional: true,
				Default:  "/",
				ForceNew: true,
			},

			"private_key": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: normalizeCert,
			},

			"name": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) > 128 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 128 characters", k))
					}
					return
				},
			},

			"name_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) > 30 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 30 characters, name is limited to 128", k))
					}
					return
				},
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

	var sslCertName string
	if v, ok := d.GetOk("name"); ok {
		sslCertName = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		sslCertName = resource.PrefixedUniqueId(v.(string))
	} else {
		sslCertName = resource.UniqueId()
	}

	createOpts := &iam.UploadServerCertificateInput{
		CertificateBody:       aws.String(d.Get("certificate_body").(string)),
		PrivateKey:            aws.String(d.Get("private_key").(string)),
		ServerCertificateName: aws.String(sslCertName),
	}

	if v, ok := d.GetOk("certificate_chain"); ok {
		createOpts.CertificateChain = aws.String(v.(string))
	}

	if v, ok := d.GetOk("path"); ok {
		createOpts.Path = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Creating IAM Server Certificate with opts: %s", createOpts)
	resp, err := conn.UploadServerCertificate(createOpts)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error uploading server certificate, error: %s: %s", awsErr.Code(), awsErr.Message())
		}
		return fmt.Errorf("[WARN] Error uploading server certificate, error: %s", err)
	}

	d.SetId(*resp.ServerCertificateMetadata.ServerCertificateId)
	d.Set("name", sslCertName)

	return resourceAwsIAMServerCertificateRead(d, meta)
}

func resourceAwsIAMServerCertificateRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	resp, err := conn.GetServerCertificate(&iam.GetServerCertificateInput{
		ServerCertificateName: aws.String(d.Get("name").(string)),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NoSuchEntity" {
				log.Printf("[WARN] IAM Server Cert (%s) not found, removing from state", d.Id())
				d.SetId("")
				return nil
			}
			return fmt.Errorf("[WARN] Error reading IAM Server Certificate: %s: %s", awsErr.Code(), awsErr.Message())
		}
		return fmt.Errorf("[WARN] Error reading IAM Server Certificate: %s", err)
	}

	// these values should always be present, and have a default if not set in
	// configuration, and so safe to reference with nil checks
	d.Set("certificate_body", normalizeCert(resp.ServerCertificate.CertificateBody))

	c := normalizeCert(resp.ServerCertificate.CertificateChain)
	if c != "" {
		d.Set("certificate_chain", c)
	}

	d.Set("path", resp.ServerCertificate.ServerCertificateMetadata.Path)
	d.Set("arn", resp.ServerCertificate.ServerCertificateMetadata.Arn)

	return nil
}

func resourceAwsIAMServerCertificateDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	log.Printf("[INFO] Deleting IAM Server Certificate: %s", d.Id())
	err := resource.Retry(3*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteServerCertificate(&iam.DeleteServerCertificateInput{
			ServerCertificateName: aws.String(d.Get("name").(string)),
		})

		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "DeleteConflict" && strings.Contains(awsErr.Message(), "currently in use by arn") {
					log.Printf("[WARN] Conflict deleting server certificate: %s, retrying", awsErr.Message())
					return resource.RetryableError(err)
				}
				if awsErr.Code() == "NoSuchEntity" {
					log.Printf("[WARN] IAM Server Certificate (%s) not found, removing from state", d.Id())
					d.SetId("")
					return nil
				}
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func normalizeCert(cert interface{}) string {
	if cert == nil || cert == (*string)(nil) {
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
