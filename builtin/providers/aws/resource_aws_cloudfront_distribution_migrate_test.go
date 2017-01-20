package aws

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestAWSCloudFrontDistributionMigrateState(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		Attributes   map[string]string
		Expected     map[string]string
		Meta         interface{}
	}{
		"v0_1_single_cache_behavior": {
			StateVersion: 0,
			Attributes: map[string]string{
				"cache_behavior.#":                                                                             "1",
				"cache_behavior.1346915471.allowed_methods.#":                                                  "2",
				"cache_behavior.1346915471.allowed_methods.0":                                                  "GET",
				"cache_behavior.1346915471.allowed_methods.1":                                                  "HEAD",
				"cache_behavior.1346915471.cached_methods.#":                                                   "2",
				"cache_behavior.1346915471.cached_methods.0":                                                   "GET",
				"cache_behavior.1346915471.cached_methods.1":                                                   "HEAD",
				"cache_behavior.1346915471.compress":                                                           "false",
				"cache_behavior.1346915471.default_ttl":                                                        "3600",
				"cache_behavior.1346915471.forwarded_values.#":                                                 "1",
				"cache_behavior.1346915471.forwarded_values.2759845635.cookies.#":                              "1",
				"cache_behavior.1346915471.forwarded_values.2759845635.cookies.2625240281.forward":             "none",
				"cache_behavior.1346915471.forwarded_values.2759845635.cookies.2625240281.whitelisted_names.#": "0",
				"cache_behavior.1346915471.forwarded_values.2759845635.headers.#":                              "0",
				"cache_behavior.1346915471.forwarded_values.2759845635.query_string":                           "false",
				"cache_behavior.1346915471.max_ttl":                                                            "86400",
				"cache_behavior.1346915471.min_ttl":                                                            "100",
				"cache_behavior.1346915471.path_pattern":                                                       "/first/*",
				"cache_behavior.1346915471.smooth_streaming":                                                   "",
				"cache_behavior.1346915471.target_origin_id":                                                   "myS3Origin",
				"cache_behavior.1346915471.trusted_signers.#":                                                  "0",
				"cache_behavior.1346915471.viewer_protocol_policy":                                             "allow-all",
			},
			Expected: map[string]string{
				"cache_behavior.#":                                                                    "1",
				"cache_behavior.0.allowed_methods.#":                                                  "2",
				"cache_behavior.0.allowed_methods.1040875975":                                         "GET",
				"cache_behavior.0.allowed_methods.1445840968":                                         "HEAD",
				"cache_behavior.0.cached_methods.#":                                                   "2",
				"cache_behavior.0.cached_methods.1040875975":                                          "GET",
				"cache_behavior.0.cached_methods.1445840968":                                          "HEAD",
				"cache_behavior.0.compress":                                                           "false",
				"cache_behavior.0.default_ttl":                                                        "3600",
				"cache_behavior.0.forwarded_values.#":                                                 "1",
				"cache_behavior.0.forwarded_values.2759845635.cookies.#":                              "1",
				"cache_behavior.0.forwarded_values.2759845635.cookies.2625240281.forward":             "none",
				"cache_behavior.0.forwarded_values.2759845635.cookies.2625240281.whitelisted_names.#": "0",
				"cache_behavior.0.forwarded_values.2759845635.headers.#":                              "0",
				"cache_behavior.0.forwarded_values.2759845635.query_string":                           "false",
				"cache_behavior.0.max_ttl":                                                            "86400",
				"cache_behavior.0.min_ttl":                                                            "100",
				"cache_behavior.0.path_pattern":                                                       "/first/*",
				"cache_behavior.0.smooth_streaming":                                                   "",
				"cache_behavior.0.target_origin_id":                                                   "myS3Origin",
				"cache_behavior.0.trusted_signers.#":                                                  "0",
				"cache_behavior.0.viewer_protocol_policy":                                             "allow-all",
			},
		},
		"v0_1_multiple_cache_behaviors": {
			StateVersion: 0,
			Attributes: map[string]string{
				"cache_behavior.#":                                                                             "2",
				"cache_behavior.1346915471.allowed_methods.#":                                                  "2",
				"cache_behavior.1346915471.allowed_methods.0":                                                  "GET",
				"cache_behavior.1346915471.allowed_methods.1":                                                  "HEAD",
				"cache_behavior.1346915471.cached_methods.#":                                                   "2",
				"cache_behavior.1346915471.cached_methods.0":                                                   "GET",
				"cache_behavior.1346915471.cached_methods.1":                                                   "HEAD",
				"cache_behavior.1346915471.compress":                                                           "false",
				"cache_behavior.1346915471.default_ttl":                                                        "3600",
				"cache_behavior.1346915471.forwarded_values.#":                                                 "1",
				"cache_behavior.1346915471.forwarded_values.2759845635.cookies.#":                              "1",
				"cache_behavior.1346915471.forwarded_values.2759845635.cookies.2625240281.forward":             "none",
				"cache_behavior.1346915471.forwarded_values.2759845635.cookies.2625240281.whitelisted_names.#": "0",
				"cache_behavior.1346915471.forwarded_values.2759845635.headers.#":                              "0",
				"cache_behavior.1346915471.forwarded_values.2759845635.query_string":                           "false",
				"cache_behavior.1346915471.max_ttl":                                                            "86400",
				"cache_behavior.1346915471.min_ttl":                                                            "100",
				"cache_behavior.1346915471.path_pattern":                                                       "/first/*",
				"cache_behavior.1346915471.smooth_streaming":                                                   "",
				"cache_behavior.1346915471.target_origin_id":                                                   "myS3Origin",
				"cache_behavior.1346915471.trusted_signers.#":                                                  "0",
				"cache_behavior.1346915471.viewer_protocol_policy":                                             "allow-all",
				"cache_behavior.2342080937.allowed_methods.#":                                                  "2",
				"cache_behavior.2342080937.allowed_methods.0":                                                  "GET",
				"cache_behavior.2342080937.allowed_methods.1":                                                  "HEAD",
				"cache_behavior.2342080937.cached_methods.#":                                                   "2",
				"cache_behavior.2342080937.cached_methods.0":                                                   "GET",
				"cache_behavior.2342080937.cached_methods.1":                                                   "HEAD",
				"cache_behavior.2342080937.compress":                                                           "false",
				"cache_behavior.2342080937.default_ttl":                                                        "3600",
				"cache_behavior.2342080937.forwarded_values.#":                                                 "1",
				"cache_behavior.2342080937.forwarded_values.2759845635.cookies.#":                              "1",
				"cache_behavior.2342080937.forwarded_values.2759845635.cookies.2625240281.forward":             "none",
				"cache_behavior.2342080937.forwarded_values.2759845635.cookies.2625240281.whitelisted_names.#": "0",
				"cache_behavior.2342080937.forwarded_values.2759845635.headers.#":                              "0",
				"cache_behavior.2342080937.forwarded_values.2759845635.query_string":                           "false",
				"cache_behavior.2342080937.max_ttl":                                                            "86400",
				"cache_behavior.2342080937.min_ttl":                                                            "200",
				"cache_behavior.2342080937.path_pattern":                                                       "/second/*",
				"cache_behavior.2342080937.smooth_streaming":                                                   "",
				"cache_behavior.2342080937.target_origin_id":                                                   "myS3Origin",
				"cache_behavior.2342080937.trusted_signers.#":                                                  "0",
				"cache_behavior.2342080937.viewer_protocol_policy":                                             "allow-all",
			},
			Expected: map[string]string{
				"cache_behavior.#":                                                                    "2",
				"cache_behavior.0.allowed_methods.#":                                                  "2",
				"cache_behavior.0.allowed_methods.1040875975":                                         "GET",
				"cache_behavior.0.allowed_methods.1445840968":                                         "HEAD",
				"cache_behavior.0.cached_methods.#":                                                   "2",
				"cache_behavior.0.cached_methods.1040875975":                                          "GET",
				"cache_behavior.0.cached_methods.1445840968":                                          "HEAD",
				"cache_behavior.0.compress":                                                           "false",
				"cache_behavior.0.default_ttl":                                                        "3600",
				"cache_behavior.0.forwarded_values.#":                                                 "1",
				"cache_behavior.0.forwarded_values.2759845635.cookies.#":                              "1",
				"cache_behavior.0.forwarded_values.2759845635.cookies.2625240281.forward":             "none",
				"cache_behavior.0.forwarded_values.2759845635.cookies.2625240281.whitelisted_names.#": "0",
				"cache_behavior.0.forwarded_values.2759845635.headers.#":                              "0",
				"cache_behavior.0.forwarded_values.2759845635.query_string":                           "false",
				"cache_behavior.0.max_ttl":                                                            "86400",
				"cache_behavior.0.min_ttl":                                                            "100",
				"cache_behavior.0.path_pattern":                                                       "/first/*",
				"cache_behavior.0.smooth_streaming":                                                   "",
				"cache_behavior.0.target_origin_id":                                                   "myS3Origin",
				"cache_behavior.0.trusted_signers.#":                                                  "0",
				"cache_behavior.0.viewer_protocol_policy":                                             "allow-all",
				"cache_behavior.1.allowed_methods.#":                                                  "2",
				"cache_behavior.1.allowed_methods.1040875975":                                         "GET",
				"cache_behavior.1.allowed_methods.1445840968":                                         "HEAD",
				"cache_behavior.1.cached_methods.#":                                                   "2",
				"cache_behavior.1.cached_methods.1040875975":                                          "GET",
				"cache_behavior.1.cached_methods.1445840968":                                          "HEAD",
				"cache_behavior.1.compress":                                                           "false",
				"cache_behavior.1.default_ttl":                                                        "3600",
				"cache_behavior.1.forwarded_values.#":                                                 "1",
				"cache_behavior.1.forwarded_values.2759845635.cookies.#":                              "1",
				"cache_behavior.1.forwarded_values.2759845635.cookies.2625240281.forward":             "none",
				"cache_behavior.1.forwarded_values.2759845635.cookies.2625240281.whitelisted_names.#": "0",
				"cache_behavior.1.forwarded_values.2759845635.headers.#":                              "0",
				"cache_behavior.1.forwarded_values.2759845635.query_string":                           "false",
				"cache_behavior.1.max_ttl":                                                            "86400",
				"cache_behavior.1.min_ttl":                                                            "200",
				"cache_behavior.1.path_pattern":                                                       "/second/*",
				"cache_behavior.1.smooth_streaming":                                                   "",
				"cache_behavior.1.target_origin_id":                                                   "myS3Origin",
				"cache_behavior.1.trusted_signers.#":                                                  "0",
				"cache_behavior.1.viewer_protocol_policy":                                             "allow-all",
			},
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         "E1L3MHGHRZSNX",
			Attributes: tc.Attributes,
		}
		migratedState, err := resourceAwsCloudFrontDistributionMigrateState(
			tc.StateVersion, is, tc.Meta)
		if err != nil {
			t.Fatalf("bad: %s, err: %#v", tn, err)
		}
		if !reflect.DeepEqual(tc.Expected, migratedState.Attributes) {
			t.Logf("[DEBUG] Diff between expected and actual: %#v", stateDiff(tc.Expected, migratedState.Attributes))
			t.Fatalf("Expected CF distribution attributes to match after migration\nGiven: %#v\nExpected: %#v",
				migratedState.Attributes, tc.Expected)
		}
	}
}

func stateDiff(expected map[string]string, actual map[string]string) map[string]string {
	var diff = make(map[string]string, 0)
	for k, v := range expected {
		if value, ok := actual[k]; !ok || value != v {
			newKey := fmt.Sprintf("%s_EXPECTED", k)
			diff[newKey] = v
		}
	}
	for k, v := range actual {
		if value, ok := expected[k]; !ok || value != v {
			newKey := fmt.Sprintf("%s_ACTUAL", k)
			diff[newKey] = v
		}
	}
	return diff
}
