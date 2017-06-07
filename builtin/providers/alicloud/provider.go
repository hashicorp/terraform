package alicloud

import (
	"github.com/denverdino/aliyungo/common"
	"github.com/hashicorp/terraform/helper/mutexkv"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"os"
)

// Provider returns a schema.Provider for alicloud
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"access_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_ACCESS_KEY", nil),
				Description: descriptions["access_key"],
			},
			"secret_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_SECRET_KEY", nil),
				Description: descriptions["secret_key"],
			},
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_REGION", DEFAULT_REGION),
				Description: descriptions["region"],
			},
		},
		DataSourcesMap: map[string]*schema.Resource{

			"alicloud_images":         dataSourceAlicloudImages(),
			"alicloud_regions":        dataSourceAlicloudRegions(),
			"alicloud_zones":          dataSourceAlicloudZones(),
			"alicloud_instance_types": dataSourceAlicloudInstanceTypes(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"alicloud_instance":                  resourceAliyunInstance(),
			"alicloud_disk":                      resourceAliyunDisk(),
			"alicloud_disk_attachment":           resourceAliyunDiskAttachment(),
			"alicloud_security_group":            resourceAliyunSecurityGroup(),
			"alicloud_security_group_rule":       resourceAliyunSecurityGroupRule(),
			"alicloud_db_instance":               resourceAlicloudDBInstance(),
			"alicloud_ess_scaling_group":         resourceAlicloudEssScalingGroup(),
			"alicloud_ess_scaling_configuration": resourceAlicloudEssScalingConfiguration(),
			"alicloud_ess_scaling_rule":          resourceAlicloudEssScalingRule(),
			"alicloud_ess_schedule":              resourceAlicloudEssSchedule(),
			"alicloud_vpc":                       resourceAliyunVpc(),
			"alicloud_nat_gateway":               resourceAliyunNatGateway(),
			//both subnet and vswith exists,cause compatible old version, and compatible aws habit.
			"alicloud_subnet":          resourceAliyunSubnet(),
			"alicloud_vswitch":         resourceAliyunSubnet(),
			"alicloud_route_entry":     resourceAliyunRouteEntry(),
			"alicloud_snat_entry":      resourceAliyunSnatEntry(),
			"alicloud_forward_entry":   resourceAliyunForwardEntry(),
			"alicloud_eip":             resourceAliyunEip(),
			"alicloud_eip_association": resourceAliyunEipAssociation(),
			"alicloud_slb":             resourceAliyunSlb(),
			"alicloud_slb_attachment":  resourceAliyunSlbAttachment(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	accesskey, ok := d.GetOk("access_key")
	if !ok {
		accesskey = os.Getenv("ALICLOUD_ACCESS_KEY")
	}
	secretkey, ok := d.GetOk("secret_key")
	if !ok {
		secretkey = os.Getenv("ALICLOUD_SECRET_KEY")
	}
	region, ok := d.GetOk("region")
	if !ok {
		region = os.Getenv("ALICLOUD_REGION")
		if region == "" {
			region = DEFAULT_REGION
		}
	}

	config := Config{
		AccessKey: accesskey.(string),
		SecretKey: secretkey.(string),
		Region:    common.Region(region.(string)),
	}

	client, err := config.Client()
	if err != nil {
		return nil, err
	}

	return client, nil
}

// This is a global MutexKV for use within this plugin.
var alicloudMutexKV = mutexkv.NewMutexKV()

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"access_key": "Access key of alicloud",
		"secret_key": "Secret key of alicloud",
		"region":     "Region of alicloud",
	}
}
