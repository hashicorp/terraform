package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudhsmv2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceCloudHsm2Cluster() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceCloudHsm2ClusterRead,

		Schema: map[string]*schema.Schema{
			"cluster_id": {
				Type:     schema.TypeString,
				Required: true,
			},

			"cluster_state": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"vpc_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"security_group_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"cluster_certificates": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cluster_certificate": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"cluster_csr": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"aws_hardware_certificate": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"hsm_certificate": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"manufacturer_hardware_certificate": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"subnet_ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func dataSourceCloudHsm2ClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudhsmv2conn

	clusterId := d.Get("cluster_id").(string)
	filters := []*string{&clusterId}
	log.Printf("[DEBUG] Reading CloudHSM v2 Cluster %s", clusterId)
	result := int64(1)
	input := &cloudhsmv2.DescribeClustersInput{
		Filters: map[string][]*string{
			"clusterIds": filters,
		},
		MaxResults: &result,
	}
	state := d.Get("cluster_state").(string)
	states := []*string{&state}
	if len(state) > 0 {
		input.Filters["states"] = states
	}
	out, err := conn.DescribeClusters(input)

	if err != nil {
		return fmt.Errorf("error describing CloudHSM v2 Cluster: %s", err)
	}

	var cluster *cloudhsmv2.Cluster
	for _, c := range out.Clusters {
		if aws.StringValue(c.ClusterId) == clusterId {
			cluster = c
			break
		}
	}

	if cluster == nil {
		return fmt.Errorf("cluster with id %s not found", clusterId)
	}

	d.SetId(clusterId)
	d.Set("vpc_id", cluster.VpcId)
	d.Set("security_group_id", cluster.SecurityGroup)
	d.Set("cluster_state", cluster.State)
	if err := d.Set("cluster_certificates", readCloudHsm2ClusterCertificates(cluster)); err != nil {
		return fmt.Errorf("error setting cluster_certificates: %s", err)
	}

	var subnets []string
	for _, sn := range cluster.SubnetMapping {
		subnets = append(subnets, *sn)
	}

	if err := d.Set("subnet_ids", subnets); err != nil {
		return fmt.Errorf("[DEBUG] Error saving Subnet IDs to state for CloudHSM v2 Cluster (%s): %s", d.Id(), err)
	}

	return nil
}
