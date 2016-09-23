package pagerduty

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"pagerduty": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProviderImpl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("PAGERDUTY_TOKEN"); v == "" {
		t.Fatal("PAGERDUTY_TOKEN must be set for acceptance tests")
	}

	if v := os.Getenv("PAGERDUTY_USER_ID"); v == "" {
		t.Fatal("PAGERDUTY_USER_ID must be set for acceptance tests")
	}

	if v := os.Getenv("PAGERDUTY_ESCALATION_POLICY_ID"); v == "" {
		t.Fatal("PAGERDUTY_ESCALATION_POLICY_ID must be set for acceptance tests")
	}
}

var importEscalationPolicyID = os.Getenv("PAGERDUTY_ESCALATION_POLICY_ID")
var importUserID = os.Getenv("PAGERDUTY_USER_ID")
var userID = os.Getenv("PAGERDUTY_USER_ID")
var escalationPolicyID = os.Getenv("PAGERDUTY_ESCALATION_POLICY_ID")
