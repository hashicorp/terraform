terraform {
  backend "local" {
    path = "imported.tfstate"
  }
}

provider "test" {
  foo = "bar"
}

resource "test_instance" "foo" {
}

resource "unknown_instance" "baz" {
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
    unknown = {
      source = "hashicorp/unknown"
    }
  }
}
