package aws

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

// See # http://docs.aws.amazon.com/general/latest/gr/rande.html#elasticbeanstalk_region
var elasticBeanstalkHostedZoneIds = map[string]string{
	"ap-southeast-1": "Z16FZ9L249IFLT",
	"ap-southeast-2": "Z2PCDNR3VC2G1N",
	"ap-northeast-1": "Z1R25G3KIG2GBW",
	"ap-northeast-2": "Z3JE5OI70TWKCP",
	"ap-south-1":     "Z18NTBI3Y7N9TZ",
	"ca-central-1":   "ZJFCZL7SSZB5I",
	"eu-central-1":   "Z1FRNW7UH4DEZJ",
	"eu-west-1":      "Z2NYPWQ7DFZAZH",
	"eu-west-2":      "Z1GKAAAUGATPF1",
	"eu-west-3":      "Z5WN6GAYWG5OB",
	"sa-east-1":      "Z10X7K2B4QSOFV",
	"us-east-1":      "Z117KPS5GTRQ2G",
	"us-east-2":      "Z14LCN19Q5QHIC",
	"us-west-1":      "Z1LQECGX5PH1X",
	"us-west-2":      "Z38NKT9BP95V3O",
}

func dataSourceAwsElasticBeanstalkHostedZone() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsElasticBeanstalkHostedZoneRead,

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func dataSourceAwsElasticBeanstalkHostedZoneRead(d *schema.ResourceData, meta interface{}) error {
	region := meta.(*AWSClient).region
	if v, ok := d.GetOk("region"); ok {
		region = v.(string)
	}

	zoneID, ok := elasticBeanstalkHostedZoneIds[region]

	if !ok {
		return fmt.Errorf("Unsupported region: %s", region)
	}

	d.SetId(zoneID)
	d.Set("region", region)
	return nil
}
