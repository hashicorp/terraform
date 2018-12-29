package aws

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestAWSCloudWatchMetricAlarmMigrateState(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		ID           string
		Attributes   map[string]string
		Expected     string
		Meta         interface{}
	}{
		"v0_1": {
			StateVersion: 0,
			ID:           "some_id",
			Attributes:   map[string]string{},
			Expected:     "missing",
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         tc.ID,
			Attributes: tc.Attributes,
		}
		is, err := resourceAwsCloudWatchMetricAlarmMigrateState(
			tc.StateVersion, is, tc.Meta)

		if err != nil {
			t.Fatalf("bad: %s, err: %#v", tn, err)
		}

		if is.Attributes["treat_missing_data"] != tc.Expected {
			t.Fatalf("bad Cloudwatch Metric Alarm Migrate: %s\n\n expected: %s", is.Attributes["treat_missing_data"], tc.Expected)
		}
	}
}
