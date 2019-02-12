package aws

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsIAMServerCertificate() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIAMServerCertificateCreate,
		Read:   resourceAwsIAMServerCertificateRead,
		Delete: resourceAwsIAMServerCertificateDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsIAMServerCertificateImport,
		},

		Schema: map[string]*schema.Schema{
			"certificate_body": {
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: normalizeCert,
			},

			"certificate_chain": {
				Type:      schema.TypeString,
				Optional:  true,
				ForceNew:  true,
				StateFunc: normalizeCert,
			},

			"path": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "/",
				ForceNew: true,
			},

			"private_key": {
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: normalizeCert,
				Sensitive: true,
			},

			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validation.StringLenBetween(0, 128),
			},

			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.StringLenBetween(0, 128-resource.UniqueIDSuffixLength),
			},

			"arn": {
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
			return fmt.Errorf("Error uploading server certificate, error: %s: %s", awsErr.Code(), awsErr.Message())
		}
		return fmt.Errorf("Error uploading server certificate, error: %s", err)
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
			return fmt.Errorf("Error reading IAM Server Certificate: %s: %s", awsErr.Code(), awsErr.Message())
		}
		return fmt.Errorf("Error reading IAM Server Certificate: %s", err)
	}

	d.SetId(*resp.ServerCertificate.ServerCertificateMetadata.ServerCertificateId)

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
	err := resource.Retry(15*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteServerCertificate(&iam.DeleteServerCertificateInput{
			ServerCertificateName: aws.String(d.Get("name").(string)),
		})

		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "DeleteConflict" && strings.Contains(awsErr.Message(), "currently in use by arn") {
					currentlyInUseBy(awsErr.Message(), meta.(*AWSClient).elbconn)
					log.Printf("[WARN] Conflict deleting server certificate: %s, retrying", awsErr.Message())
					return resource.RetryableError(err)
				}
				if awsErr.Code() == "NoSuchEntity" {
					return nil
				}
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	return err
}

func resourceAwsIAMServerCertificateImport(
	d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	d.Set("name", d.Id())
	// private_key can't be fetched from any API call
	return []*schema.ResourceData{d}, nil
}

func currentlyInUseBy(awsErr string, conn *elb.ELB) {
	r := regexp.MustCompile(`currently in use by ([a-z0-9:-]+)\/([a-z0-9-]+)\.`)
	matches := r.FindStringSubmatch(awsErr)
	if len(matches) > 0 {
		lbName := matches[2]
		describeElbOpts := &elb.DescribeLoadBalancersInput{
			LoadBalancerNames: []*string{aws.String(lbName)},
		}
		if _, err := conn.DescribeLoadBalancers(describeElbOpts); err != nil {
			if isAWSErr(err, "LoadBalancerNotFound", "") {
				log.Printf("[WARN] Load Balancer (%s) causing delete conflict not found", lbName)
			}
		}
	}
}

func normalizeCert(cert interface{}) string {
	if cert == nil || cert == (*string)(nil) {
		return ""
	}

	var rawCert string
	switch cert.(type) {
	case string:
		rawCert = cert.(string)
	case *string:
		rawCert = *cert.(*string)
	default:
		return ""
	}

	cleanVal := sha1.Sum(stripCR([]byte(strings.TrimSpace(rawCert))))
	return hex.EncodeToString(cleanVal[:])
}

// strip CRs from raw literals. Lifted from go/scanner/scanner.go
// See https://github.com/golang/go/blob/release-branch.go1.6/src/go/scanner/scanner.go#L479
func stripCR(b []byte) []byte {
	c := make([]byte, len(b))
	i := 0
	for _, ch := range b {
		if ch != '\r' {
			c[i] = ch
			i++
		}
	}
	return c[:i]
}
