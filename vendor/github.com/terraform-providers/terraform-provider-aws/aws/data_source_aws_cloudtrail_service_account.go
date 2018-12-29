package aws

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

// See http://docs.aws.amazon.com/awscloudtrail/latest/userguide/cloudtrail-supported-regions.html
var cloudTrailServiceAccountPerRegionMap = map[string]string{
	"us-east-1":      "086441151436",
	"us-east-2":      "475085895292",
	"us-west-1":      "388731089494",
	"us-west-2":      "113285607260",
	"ap-south-1":     "977081816279",
	"ap-northeast-2": "492519147666",
	"ap-southeast-1": "903692715234",
	"ap-southeast-2": "284668455005",
	"ap-northeast-1": "216624486486",
	"ca-central-1":   "819402241893",
	"eu-central-1":   "035351147821",
	"eu-west-1":      "859597730677",
	"eu-west-2":      "282025262664",
	"eu-west-3":      "262312530599",
	"sa-east-1":      "814480443879",
	"cn-northwest-1": "681348832753",
}

func dataSourceAwsCloudTrailServiceAccount() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsCloudTrailServiceAccountRead,

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsCloudTrailServiceAccountRead(d *schema.ResourceData, meta interface{}) error {
	region := meta.(*AWSClient).region
	if v, ok := d.GetOk("region"); ok {
		region = v.(string)
	}

	if accid, ok := cloudTrailServiceAccountPerRegionMap[region]; ok {
		d.SetId(accid)
		d.Set("arn", iamArnString(meta.(*AWSClient).partition, accid, "root"))
		return nil
	}

	return fmt.Errorf("Unknown region (%q)", region)
}
