package random

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccResourcePet_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourcePet_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccResourcePetLength("random_pet.pet_1", "-", 2),
				),
			},
		},
	})
}

func TestAccResourcePet_length(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourcePet_length,
				Check: resource.ComposeTestCheckFunc(
					testAccResourcePetLength("random_pet.pet_1", "-", 4),
				),
			},
		},
	})
}

func TestAccResourcePet_prefix(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourcePet_prefix,
				Check: resource.ComposeTestCheckFunc(
					testAccResourcePetLength("random_pet.pet_1", "-", 3),
					resource.TestMatchResourceAttr(
						"random_pet.pet_1", "id", regexp.MustCompile("^consul-")),
				),
			},
		},
	})
}

func TestAccResourcePet_separator(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourcePet_separator,
				Check: resource.ComposeTestCheckFunc(
					testAccResourcePetLength("random_pet.pet_1", "_", 3),
				),
			},
		},
	})
}

func testAccResourcePetLength(id string, separator string, length int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("Not found: %s", id)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		petParts := strings.Split(rs.Primary.ID, separator)
		if len(petParts) != length {
			return fmt.Errorf("Length does not match")
		}

		return nil
	}
}

const testAccResourcePet_basic = `
resource "random_pet" "pet_1" {
}
`

const testAccResourcePet_length = `
resource "random_pet" "pet_1" {
  length = 4
}
`
const testAccResourcePet_prefix = `
resource "random_pet" "pet_1" {
  prefix = "consul"
}
`

const testAccResourcePet_separator = `
resource "random_pet" "pet_1" {
  length = 3
  separator = "_"
}
`
