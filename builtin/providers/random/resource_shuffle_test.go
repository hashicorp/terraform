package random

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccResourceShuffle(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccResourceShuffleConfig,
				Check: resource.ComposeTestCheckFunc(
					// These results are current as of Go 1.6. The Go
					// "rand" package does not guarantee that the random
					// number generator will generate the same results
					// forever, but the maintainers endeavor not to change
					// it gratuitously.
					// These tests allow us to detect such changes and
					// document them when they arise, but the docs for this
					// resource specifically warn that results are not
					// guaranteed consistent across Terraform releases.
					testAccResourceShuffleCheck(
						"random_shuffle.default_length",
						[]string{"a", "c", "b", "e", "d"},
					),
					testAccResourceShuffleCheck(
						"random_shuffle.shorter_length",
						[]string{"a", "c", "b"},
					),
					testAccResourceShuffleCheck(
						"random_shuffle.longer_length",
						[]string{"a", "c", "b", "e", "d", "a", "e", "d", "c", "b", "a", "b"},
					),
				),
			},
		},
	})
}

func testAccResourceShuffleCheck(id string, wants []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("Not found: %s", id)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		attrs := rs.Primary.Attributes

		gotLen := attrs["result.#"]
		wantLen := strconv.Itoa(len(wants))
		if gotLen != wantLen {
			return fmt.Errorf("got %s result items; want %s", gotLen, wantLen)
		}

		for i, want := range wants {
			key := fmt.Sprintf("result.%d", i)
			if got := attrs[key]; got != want {
				return fmt.Errorf("index %d is %q; want %q", i, got, want)
			}
		}

		return nil
	}
}

const testAccResourceShuffleConfig = `
resource "random_shuffle" "default_length" {
    input = ["a", "b", "c", "d", "e"]
    seed = "-"
}
resource "random_shuffle" "shorter_length" {
    input = ["a", "b", "c", "d", "e"]
    seed = "-"
    result_count = 3
}
resource "random_shuffle" "longer_length" {
    input = ["a", "b", "c", "d", "e"]
    seed = "-"
    result_count = 12
}
`
