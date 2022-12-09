terraform {
  required_providers {
    null = "~> 2.0.0"
    random = {
      version = "~> 1.2.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 3.0"
    }
  }
}

# There is no provider in required_providers called "http", so this
# implicitly declares a dependency on "hashicorp/http".
resource "http_foo" "bar" {
}

module "child" {
  source = "./child"
}
