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

		gotCount, _ := strconv.Atoi(count)
		if gotCount != len(value) {
			return fmt.Errorf("Mismatch array count for %s: got %s, wanted %d", key, count, len(value))
		}

	Next:
		for i := 0; i < gotCount; i++ {
			attrKey = fmt.Sprintf("%s.%d", key, i)
			got, ok := is.Attributes[attrKey]
			if !ok {
				return fmt.Errorf("Missing array item for %s", attrKey)
			}
			for _, want := range value {
				if got == want {
					continue Next
				}
			}
			return fmt.Errorf(
				"Unexpected array item for %s: got %s",
				attrKey,
				got)
		}

		return nil
	}
}
