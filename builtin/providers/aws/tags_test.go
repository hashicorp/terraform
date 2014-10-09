package aws

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

// testAccCheckTags can be used to check the tags on a resource.
func testAccCheckTags(
	ts *[]ec2.Tag, key string, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		m := tagsToMap(*ts)
		v, ok := m[key]
		if !ok {
			return fmt.Errorf("Missing tag: %s", key)
		}
		if v != value {
			return fmt.Errorf("%s: bad value: %s", key, v)
		}

		return nil
	}
}
