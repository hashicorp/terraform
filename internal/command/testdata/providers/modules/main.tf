terraform {
  required_providers {
    foo = {
      version = "1.0"
    }
  }
}

provider "bar" {
  version = "2.0.0"
}

module "kiddo" {
  source = "./child"
}
