package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func dataSourceAwsEksCluster() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsEksClusterRead,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"certificate_authority": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"data": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"role_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vpc_config": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"security_group_ids": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"subnet_ids": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"vpc_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"version": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsEksClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).eksconn
	name := d.Get("name").(string)

	input := &eks.DescribeClusterInput{
		Name: aws.String(name),
	}

	log.Printf("[DEBUG] Reading EKS Cluster: %s", input)
	output, err := conn.DescribeCluster(input)
	if err != nil {
		return fmt.Errorf("error reading EKS Cluster (%s): %s", name, err)
	}

	cluster := output.Cluster
	if cluster == nil {
		return fmt.Errorf("EKS Cluster (%s) not found", name)
	}

	d.SetId(name)
	d.Set("arn", cluster.Arn)

	if err := d.Set("certificate_authority", flattenEksCertificate(cluster.CertificateAuthority)); err != nil {
		return fmt.Errorf("error setting certificate_authority: %s", err)
	}

	d.Set("created_at", aws.TimeValue(cluster.CreatedAt).String())
	d.Set("endpoint", cluster.Endpoint)
	d.Set("name", cluster.Name)
	d.Set("role_arn", cluster.RoleArn)
	d.Set("version", cluster.Version)

	if err := d.Set("vpc_config", flattenEksVpcConfigResponse(cluster.ResourcesVpcConfig)); err != nil {
		return fmt.Errorf("error setting vpc_config: %s", err)
	}

	return nil
}
