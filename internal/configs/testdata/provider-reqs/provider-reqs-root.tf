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

# There is no provider in required_providers called "vault", so this
# implicitly declares a dependency on "hashicorp/vault".
resource "vault_foo" "bar" {
}

module "kinder" {
  source = "./child"
}

# There is no provider in required_providers called "terraform", but for
# this name in particular we imply terraform.io/builtin/terraform instead,
# to avoid selecting the now-unmaintained
# registry.terraform.io/hashicorp/terraform.
data "terraform_remote_state" "bar" {
}

# There is no provider in required_providers called "http", so the version
# constraint should come from this configuration block.
provider "http" {
  version = "~> 1.4"
}
