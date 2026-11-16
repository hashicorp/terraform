required_providers {
  other = {
    source  = "hashicorp/other"
    version = "0.1.0"
  }
}

provider "other" "main" {}

component "self" {
  source = "./"

  providers = {
    other = provider.other.main
  }
}
