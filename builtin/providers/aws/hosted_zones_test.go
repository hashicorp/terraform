package aws

import (
	"testing"
)

func TestHostedZoneIDForRegion(t *testing.T) {
	if r := HostedZoneIDForRegion("us-east-1"); r != "Z3AQBSTGFYJSTF" {
		t.Fatalf("bad: %s", r)
	}
	if r := HostedZoneIDForRegion("ap-southeast-2"); r != "Z1WCIGYICN2BYD" {
		t.Fatalf("bad: %s", r)
	}

	// Empty string should default to us-east-1
	if r := HostedZoneIDForRegion(""); r != "Z3AQBSTGFYJSTF" {
		t.Fatalf("bad: %s", r)
	}

	// Bad input should be empty string
	if r := HostedZoneIDForRegion("not-a-region"); r != "" {
		t.Fatalf("bad: %s", r)
	}
}
