package aws

import "testing"

func TestWebsiteEndpointUrl_withoutRegion(t *testing.T) {
	u := WebsiteEndpointUrl("buck.et", "")
	if u != "buck.et.s3-website-us-east-1.amazonaws.com" {
		t.Fatalf("bad: %s", u)
	}
}

func TestWebsiteEndpointUrl_withRegion(t *testing.T) {
	u := WebsiteEndpointUrl("buck.et", "us-west-1")
	if u != "buck.et.s3-website-us-west-1.amazonaws.com" {
		t.Fatalf("bad: %s", u)
	}
}
