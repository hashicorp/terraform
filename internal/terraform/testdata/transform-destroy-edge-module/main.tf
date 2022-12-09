terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

resource "test_object" "a" {
  test_string = "${module.child.output}"
}

module "child" {
  source = "./child"
}
