package aws

import (
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/resource"
)

// resourceMap is the mapping of resources we support to their basic
// operations. This makes it easy to implement new resource types.
var resourceMap *resource.Map

func init() {
	resourceMap = &resource.Map{
		Mapping: map[string]resource.Resource{
			"aws_autoscaling_group": resource.Resource{
				ConfigValidator: resource_aws_autoscaling_group_validation(),
				Create:          resource_aws_autoscaling_group_create,
				Destroy:         resource_aws_autoscaling_group_destroy,
				Diff:            resource_aws_autoscaling_group_diff,
				Refresh:         resource_aws_autoscaling_group_refresh,
				Update:          resource_aws_autoscaling_group_update,
			},

			"aws_db_instance": resource.Resource{
				ConfigValidator: resource_aws_db_instance_validation(),
				Create:          resource_aws_db_instance_create,
				Destroy:         resource_aws_db_instance_destroy,
				Diff:            resource_aws_db_instance_diff,
				Refresh:         resource_aws_db_instance_refresh,
				Update:          resource_aws_db_instance_update,
			},

			"aws_db_security_group": resource.Resource{
				ConfigValidator: resource_aws_db_security_group_validation(),
				Create:          resource_aws_db_security_group_create,
				Destroy:         resource_aws_db_security_group_destroy,
				Diff:            resource_aws_db_security_group_diff,
				Refresh:         resource_aws_db_security_group_refresh,
			},

			"aws_eip": resource.Resource{
				ConfigValidator: resource_aws_eip_validation(),
				Create:          resource_aws_eip_create,
				Destroy:         resource_aws_eip_destroy,
				Diff:            resource_aws_eip_diff,
				Refresh:         resource_aws_eip_refresh,
				Update:          resource_aws_eip_update,
			},

			"aws_elb": resource.Resource{
				ConfigValidator: resource_aws_elb_validation(),
				Create:          resource_aws_elb_create,
				Update:          resource_aws_elb_update,
				Destroy:         resource_aws_elb_destroy,
				Diff:            resource_aws_elb_diff,
				Refresh:         resource_aws_elb_refresh,
			},

			"aws_instance": resource.Resource{
				Create:  resource_aws_instance_create,
				Destroy: resource_aws_instance_destroy,
				Diff:    resource_aws_instance_diff,
				Refresh: resource_aws_instance_refresh,
				Update:  resource_aws_instance_update,
			},

			"aws_internet_gateway": resource.Resource{
				Create:  resource_aws_internet_gateway_create,
				Destroy: resource_aws_internet_gateway_destroy,
				Diff:    resource_aws_internet_gateway_diff,
				Refresh: resource_aws_internet_gateway_refresh,
				Update:  resource_aws_internet_gateway_update,
			},

			"aws_launch_configuration": resource.Resource{
				ConfigValidator: resource_aws_launch_configuration_validation(),
				Create:          resource_aws_launch_configuration_create,
				Destroy:         resource_aws_launch_configuration_destroy,
				Diff:            resource_aws_launch_configuration_diff,
				Refresh:         resource_aws_launch_configuration_refresh,
			},

			"aws_route_table": resource.Resource{
				ConfigValidator: &config.Validator{
					Required: []string{
						"vpc_id",
						"route.*.cidr_block",
					},
					Optional: []string{
						"route.*.gateway_id",
						"route.*.instance_id",
					},
				},
				Create:  resource_aws_route_table_create,
				Destroy: resource_aws_route_table_destroy,
				Diff:    resource_aws_route_table_diff,
				Refresh: resource_aws_route_table_refresh,
				Update:  resource_aws_route_table_update,
			},

			"aws_route_table_association": resource.Resource{
				ConfigValidator: &config.Validator{
					Required: []string{
						"route_table_id",
						"subnet_id",
					},
				},
				Create:  resource_aws_route_table_association_create,
				Destroy: resource_aws_route_table_association_destroy,
				Diff:    resource_aws_route_table_association_diff,
				Refresh: resource_aws_route_table_association_refresh,
				Update:  resource_aws_route_table_association_update,
			},

			"aws_route53_record": resource.Resource{
				ConfigValidator: resource_aws_r53_record_validation(),
				Create:          resource_aws_r53_record_create,
				Destroy:         resource_aws_r53_record_destroy,
				Diff:            resource_aws_r53_record_diff,
				Refresh:         resource_aws_r53_record_refresh,
				Update:          resource_aws_r53_record_create,
			},

			"aws_route53_zone": resource.Resource{
				ConfigValidator: resource_aws_r53_zone_validation(),
				Create:          resource_aws_r53_zone_create,
				Destroy:         resource_aws_r53_zone_destroy,
				Diff:            resource_aws_r53_zone_diff,
				Refresh:         resource_aws_r53_zone_refresh,
			},

			"aws_s3_bucket": resource.Resource{
				ConfigValidator: resource_aws_s3_bucket_validation(),
				Create:          resource_aws_s3_bucket_create,
				Destroy:         resource_aws_s3_bucket_destroy,
				Diff:            resource_aws_s3_bucket_diff,
				Refresh:         resource_aws_s3_bucket_refresh,
			},

			"aws_security_group": resource.Resource{
				ConfigValidator: resource_aws_security_group_validation(),
				Create:          resource_aws_security_group_create,
				Destroy:         resource_aws_security_group_destroy,
				Diff:            resource_aws_security_group_diff,
				Refresh:         resource_aws_security_group_refresh,
			},

			"aws_subnet": resource.Resource{
				Create:  resource_aws_subnet_create,
				Destroy: resource_aws_subnet_destroy,
				Diff:    resource_aws_subnet_diff,
				Refresh: resource_aws_subnet_refresh,
			},

			"aws_vpc": resource.Resource{
				Create:  resource_aws_vpc_create,
				Destroy: resource_aws_vpc_destroy,
				Diff:    resource_aws_vpc_diff,
				Refresh: resource_aws_vpc_refresh,
			},
		},
	}
}
