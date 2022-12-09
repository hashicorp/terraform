terraform {
  required_providers {
    null = {
      version = "1.0"
    }
  }
}

provider "http" {
  version = "2.0.0"
}

module "kiddo" {
  source = "./child"
}
