package aws

import "fmt"

// This list is copied from
// http://docs.aws.amazon.com/general/latest/gr/rande.html#s3_website_region_endpoints
// It currently cannot be generated from the API json.
var hostedZoneIDsMap = map[string]string{
	"ap-northeast-1": "Z2M4EHUR26P7ZW",
	"ap-northeast-2": "Z3W03O7B5YMIYP",
	"ap-northeast-3": "Z2YQB5RD63NC85",
	"ap-south-1":     "Z11RGJOFQNVJUP",
	"ap-southeast-1": "Z3O0J2DXBE1FTB",
	"ap-southeast-2": "Z1WCIGYICN2BYD",
	"ca-central-1":   "Z1QDHH18159H29",
	"eu-central-1":   "Z21DNDUVLTQW6Q",
	"eu-north-1":     "Z3BAZG2TWCNX0D",
	"eu-west-1":      "Z1BKCTXD74EZPE",
	"eu-west-2":      "Z3GKZC51ZF0DB4",
	"eu-west-3":      "Z3R1K369G5AVDG",
	"sa-east-1":      "Z7KQH4QJS55SO",
	"us-east-1":      "Z3AQBSTGFYJSTF",
	"us-east-2":      "Z2O1EMRO9K5GLX",
	"us-gov-east-1":  "Z31GFT0UA1I2HV",
	"us-gov-west-1":  "Z31GFT0UA1I2HV",
	"us-west-1":      "Z2F56UZL2M1ACD",
	"us-west-2":      "Z3BJ6K6RIION7M",
}

// Returns the hosted zone ID for an S3 website endpoint region. This can be
// used as input to the aws_route53_record resource's zone_id argument.
func HostedZoneIDForRegion(region string) (string, error) {
	if v, ok := hostedZoneIDsMap[region]; ok {
		return v, nil
	}
	return "", fmt.Errorf("S3 hosted zone ID not found for region: %s", region)
}
