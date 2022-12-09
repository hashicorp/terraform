provider "test" {
	version = "1.2.3"
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
