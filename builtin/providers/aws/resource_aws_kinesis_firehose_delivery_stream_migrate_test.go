package aws

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestAWSKinesisFirehoseMigrateState(t *testing.T) {
	cases := map[string]struct {
		StateVersion int
		Attributes   map[string]string
		Expected     map[string]string
		Meta         interface{}
	}{
		"v0.6.16 and earlier": {
			StateVersion: 0,
			Attributes: map[string]string{
				// EBS
				"role_arn":            "arn:aws:iam::somenumber:role/tf_acctest_4271506651559170635",
				"s3_bucket_arn":       "arn:aws:s3:::tf-test-bucket",
				"s3_buffer_interval":  "400",
				"s3_buffer_size":      "10",
				"s3_data_compression": "GZIP",
			},
			Expected: map[string]string{
				"s3_configuration.#":                    "1",
				"s3_configuration.0.bucket_arn":         "arn:aws:s3:::tf-test-bucket",
				"s3_configuration.0.buffer_interval":    "400",
				"s3_configuration.0.buffer_size":        "10",
				"s3_configuration.0.compression_format": "GZIP",
				"s3_configuration.0.role_arn":           "arn:aws:iam::somenumber:role/tf_acctest_4271506651559170635",
			},
		},
		"v0.6.16 and earlier, sparse": {
			StateVersion: 0,
			Attributes: map[string]string{
				// EBS
				"role_arn":      "arn:aws:iam::somenumber:role/tf_acctest_4271506651559170635",
				"s3_bucket_arn": "arn:aws:s3:::tf-test-bucket",
			},
			Expected: map[string]string{
				"s3_configuration.#":            "1",
				"s3_configuration.0.bucket_arn": "arn:aws:s3:::tf-test-bucket",
				"s3_configuration.0.role_arn":   "arn:aws:iam::somenumber:role/tf_acctest_4271506651559170635",
			},
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         "i-abc123",
			Attributes: tc.Attributes,
		}
		is, err := resourceAwsKinesisFirehoseMigrateState(
			tc.StateVersion, is, tc.Meta)

		if err != nil {
			t.Fatalf("bad: %s, err: %#v", tn, err)
		}

		for k, v := range tc.Expected {
			if is.Attributes[k] != v {
				t.Fatalf(
					"bad: %s\n\n expected: %#v -> %#v\n got: %#v -> %#v\n in: %#v",
					tn, k, v, k, is.Attributes[k], is.Attributes)
			}
		}
	}
}

func TestAWSKinesisFirehoseMigrateState_empty(t *testing.T) {
	var is *terraform.InstanceState
	var meta interface{}

	// should handle nil
	is, err := resourceAwsKinesisFirehoseMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
	if is != nil {
		t.Fatalf("expected nil instancestate, got: %#v", is)
	}

	// should handle non-nil but empty
	is = &terraform.InstanceState{}
	is, err = resourceAwsInstanceMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
}
