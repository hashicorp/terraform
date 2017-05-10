package dns

import (
	"fmt"

	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func testCheckAttrStringArrayMember(name, key string, value []string) r.TestCheckFunc {
	return func(s *terraform.State) error {
		ms := s.RootModule()
		rs, ok := ms.Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		is := rs.Primary
		if is == nil {
			return fmt.Errorf("No primary instance: %s", name)
		}

		got, ok := is.Attributes[key]
		if !ok {
			return fmt.Errorf("Attributes not found for %s", key)
		}

		for _, want := range value {
			if got == want {
				return nil
			}
		}

		return fmt.Errorf(
			"Unexpected value for %s: got %s",
			key,
			got)
	}
}
