package aws

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsIAMServerCertificate() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsIAMServerCertificateRead,

		Schema: map[string]*schema.Schema{
			"name": {
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

			"name_prefix": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) > 102 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 102 characters, name is limited to 128", k))
					}
					return
				},
			},

			"latest": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"path": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"expiration_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

type certificateByExpiration []*iam.ServerCertificateMetadata

func (m certificateByExpiration) Len() int {
	return len(m)
}

func (m certificateByExpiration) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m certificateByExpiration) Less(i, j int) bool {
	return m[i].Expiration.After(*m[j].Expiration)
}

func dataSourceAwsIAMServerCertificateRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	var matcher = func(cert *iam.ServerCertificateMetadata) bool {
		return strings.HasPrefix(aws.StringValue(cert.ServerCertificateName), d.Get("name_prefix").(string))
	}
	if v, ok := d.GetOk("name"); ok {
		matcher = func(cert *iam.ServerCertificateMetadata) bool {
			return aws.StringValue(cert.ServerCertificateName) == v.(string)
		}
	}

	var metadatas = []*iam.ServerCertificateMetadata{}
	err := iamconn.ListServerCertificatesPages(&iam.ListServerCertificatesInput{}, func(p *iam.ListServerCertificatesOutput, lastPage bool) bool {
		for _, cert := range p.ServerCertificateMetadataList {
			if matcher(cert) {
				metadatas = append(metadatas, cert)
			}
		}
		return true
	})
	if err != nil {
		return errwrap.Wrapf("Error describing certificates: {{err}}", err)
	}

	if len(metadatas) == 0 {
		return fmt.Errorf("Search for AWS IAM server certificate returned no results")
	}
	if len(metadatas) > 1 {
		if !d.Get("latest").(bool) {
			return fmt.Errorf("Search for AWS IAM server certificate returned too many results")
		}

		sort.Sort(certificateByExpiration(metadatas))
	}

	metadata := metadatas[0]
	d.SetId(*metadata.ServerCertificateId)
	d.Set("arn", *metadata.Arn)
	d.Set("path", *metadata.Path)
	d.Set("name", *metadata.ServerCertificateName)
	if metadata.Expiration != nil {
		d.Set("expiration_date", metadata.Expiration.Format("2006-01-02T15:04:05"))
	}

	return nil
}
