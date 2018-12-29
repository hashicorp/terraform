package aws

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

// See http://docs.aws.amazon.com/general/latest/gr/rande.html#elb_region
var elbHostedZoneIdPerRegionMap = map[string]string{
	"ap-northeast-1": "Z14GRHDCWA56QT",
	"ap-northeast-2": "ZWKZPGTI48KDX",
	"ap-northeast-3": "Z5LXEXXYW11ES",
	"ap-south-1":     "ZP97RAFLXTNZK",
	"ap-southeast-1": "Z1LMS91P8CMLE5",
	"ap-southeast-2": "Z1GM3OXH4ZPM65",
	"ca-central-1":   "ZQSVJUPU6J1EY",
	"cn-north-1":     "638102146993",
	"eu-central-1":   "Z215JYRZR1TBD5",
	"eu-north-1":     "Z23TAZ6LKFMNIO",
	"eu-west-1":      "Z32O12XQLNTSW2",
	"eu-west-2":      "ZHURV8PSTC4K8",
	"eu-west-3":      "Z3Q77PNBQS71R4",
	"sa-east-1":      "Z2P70J7HTTTPLU",
	"us-east-1":      "Z35SXDOTRQ7X7K",
	"us-east-2":      "Z3AADJGX6KTTL2",
	"us-gov-west-1":  "048591011584",
	"us-west-1":      "Z368ELLRRE2KJ0",
	"us-west-2":      "Z1H1FL5HABSF5",
}

func dataSourceAwsElbHostedZoneId() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsElbHostedZoneIdRead,

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func dataSourceAwsElbHostedZoneIdRead(d *schema.ResourceData, meta interface{}) error {
	region := meta.(*AWSClient).region
	if v, ok := d.GetOk("region"); ok {
		region = v.(string)
	}

	if zoneId, ok := elbHostedZoneIdPerRegionMap[region]; ok {
		d.SetId(zoneId)
		return nil
	}

	return fmt.Errorf("Unknown region (%q)", region)
}
