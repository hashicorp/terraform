terraform {
  required_providers {
    nonexist = {
      source = "terraform.io/builtin/nonexist"
    }
    terraform = {
      version = "1.2.0"
    }
  }
}
