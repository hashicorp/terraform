package dns

import (
	"fmt"
	"strconv"

	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func testCheckAttrStringArray(name, key string, value []string) r.TestCheckFunc {
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

		attrKey := fmt.Sprintf("%s.#", key)
		count, ok := is.Attributes[attrKey]
		if !ok {
			return fmt.Errorf("Attributes not found for %s", attrKey)
		}

		got, _ := strconv.Atoi(count)
		if got != len(value) {
			return fmt.Errorf("Mismatch array count for %s: got %s, wanted %d", key, count, len(value))
		}

		for i, want := range value {
			attrKey = fmt.Sprintf("%s.%d", key, i)
			got, ok := is.Attributes[attrKey]
			if !ok {
				return fmt.Errorf("Missing array item for %s", attrKey)
			}
			if got != want {
				return fmt.Errorf(
					"Mismatched array item for %s: got %s, want %s",
					attrKey,
					got,
					want)
			}
		}

		return nil
	}
}
