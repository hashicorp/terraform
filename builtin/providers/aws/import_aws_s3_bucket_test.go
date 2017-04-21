package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSS3Bucket_importBasic(t *testing.T) {
	resourceName := "aws_s3_bucket.bucket"
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSS3BucketConfig(rInt),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"force_destroy", "acl"},
			},
		},
	})
}

func TestAccAWSS3Bucket_importWithPolicy(t *testing.T) {
	rInt := acctest.RandInt()

	checkFn := func(s []*terraform.InstanceState) error {
		// Expect 2: bucket + policy
		if len(s) != 2 {
			return fmt.Errorf("expected 2 states: %#v", s)
		}
		bucketState, policyState := s[0], s[1]

		expectedBucketId := fmt.Sprintf("tf-test-bucket-%d", rInt)

		if bucketState.ID != expectedBucketId {
			return fmt.Errorf("expected bucket of ID %s, %s received",
				expectedBucketId, bucketState.ID)
		}

		if policyState.ID != expectedBucketId {
			return fmt.Errorf("expected policy of ID %s, %s received",
				expectedBucketId, bucketState.ID)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSS3BucketConfigWithPolicy(rInt),
			},

			{
				ResourceName:     "aws_s3_bucket.bucket",
				ImportState:      true,
				ImportStateCheck: checkFn,
			},
		},
	})
}
