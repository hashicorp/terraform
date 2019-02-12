package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/hashicorp/terraform/helper/schema"
)

// See http://docs.aws.amazon.com/awscloudtrail/latest/userguide/cloudtrail-supported-regions.html
// See https://docs.aws.amazon.com/govcloud-us/latest/ug-east/verifying-cloudtrail.html
// See https://docs.aws.amazon.com/govcloud-us/latest/ug-west/verifying-cloudtrail.html
var cloudTrailServiceAccountPerRegionMap = map[string]string{
	"ap-northeast-1": "216624486486",
	"ap-northeast-2": "492519147666",
	"ap-northeast-3": "765225791966",
	"ap-south-1":     "977081816279",
	"ap-southeast-1": "903692715234",
	"ap-southeast-2": "284668455005",
	"ca-central-1":   "819402241893",
	"cn-northwest-1": "681348832753",
	"eu-central-1":   "035351147821",
	"eu-north-1":     "829690693026",
	"eu-west-1":      "859597730677",
	"eu-west-2":      "282025262664",
	"eu-west-3":      "262312530599",
	"sa-east-1":      "814480443879",
	"us-east-1":      "086441151436",
	"us-east-2":      "475085895292",
	"us-gov-east-1":  "608710470296",
	"us-gov-west-1":  "608710470296",
	"us-west-1":      "388731089494",
	"us-west-2":      "113285607260",
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
		arn := arn.ARN{
			Partition: meta.(*AWSClient).partition,
			Service:   "iam",
			AccountID: accid,
			Resource:  "root",
		}.String()
		d.Set("arn", arn)

		return nil
	}

	return fmt.Errorf("Unknown region (%q)", region)
}
