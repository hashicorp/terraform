package aws

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

// See http://docs.aws.amazon.com/redshift/latest/mgmt/db-auditing.html#db-auditing-enable-logging
var redshiftServiceAccountPerRegionMap = map[string]string{
	"us-east-1":      "193672423079",
	"us-east-2":      "391106570357",
	"us-west-1":      "262260360010",
	"us-west-2":      "902366379725",
	"ap-south-1":     "865932855811",
	"ap-northeast-2": "760740231472",
	"ap-southeast-1": "361669875840",
	"ap-southeast-2": "762762565011",
	"ap-northeast-1": "404641285394",
	"ca-central-1":   "907379612154",
	"eu-central-1":   "053454850223",
	"eu-west-1":      "210876761215",
	"eu-west-2":      "307160386991",
	"sa-east-1":      "075028567923",
}

func dataSourceAwsRedshiftServiceAccount() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsRedshiftServiceAccountRead,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
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

func dataSourceAwsRedshiftServiceAccountRead(d *schema.ResourceData, meta interface{}) error {
	region := meta.(*AWSClient).region
	if v, ok := d.GetOk("region"); ok {
		region = v.(string)
	}

	if accid, ok := redshiftServiceAccountPerRegionMap[region]; ok {
		d.SetId(accid)
		d.Set("arn", iamArnString(meta.(*AWSClient).partition, accid, "user/logs"))
		return nil
	}

	return fmt.Errorf("Unknown region (%q)", region)
}
