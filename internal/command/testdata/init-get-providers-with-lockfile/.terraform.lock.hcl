# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/exact" {
  version     = "1.2.3"
  constraints = "1.2.3"
}

provider "registry.terraform.io/hashicorp/greater-than" {
  version     = "2.3.3"
  constraints = ">= 2.3.3"
}

provider "registry.terraform.io/hashicorp/between" {
  version     = "2.3.4"
  constraints = "> 1.0.0, < 3.0.0"
}
