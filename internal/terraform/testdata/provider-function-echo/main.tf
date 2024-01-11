terraform {
  required_providers {
    test = {
      source = "registry.terraform.io/hashicorp/test"
	}
  }
}

resource "test_object" "test" {
  test_string = provider::test::echo("input")
}
