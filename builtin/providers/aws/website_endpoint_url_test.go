package aws

import "testing"

// http://docs.aws.amazon.com/AmazonS3/latest/dev/WebsiteEndpoints.html
var websiteEndpoints = []struct {
	in  string
	out string
}{
	{"", "bucket-name.s3-website-us-east-1.amazonaws.com"},
	{"us-west-2", "bucket-name.s3-website-us-west-2.amazonaws.com"},
	{"us-west-1", "bucket-name.s3-website-us-west-1.amazonaws.com"},
	{"eu-west-1", "bucket-name.s3-website-eu-west-1.amazonaws.com"},
	{"eu-central-1", "bucket-name.s3-website.eu-central-1.amazonaws.com"},
	{"ap-south-1", "bucket-name.s3-website.ap-south-1.amazonaws.com"},
	{"ap-southeast-1", "bucket-name.s3-website-ap-southeast-1.amazonaws.com"},
	{"ap-northeast-1", "bucket-name.s3-website-ap-northeast-1.amazonaws.com"},
	{"ap-southeast-2", "bucket-name.s3-website-ap-southeast-2.amazonaws.com"},
	{"ap-northeast-2", "bucket-name.s3-website.ap-northeast-2.amazonaws.com"},
	{"sa-east-1", "bucket-name.s3-website-sa-east-1.amazonaws.com"},
}

func TestWebsiteEndpointUrl(t *testing.T) {
	for _, tt := range websiteEndpoints {
		s := WebsiteEndpoint("bucket-name", tt.in)
		if s.Endpoint != tt.out {
			t.Errorf("WebsiteEndpointUrl(\"bucket-name\", %q) => %q, want %q", tt.in, s.Endpoint, tt.out)
		}
	}
}
