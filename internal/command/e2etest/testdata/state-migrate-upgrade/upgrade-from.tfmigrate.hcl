state_store_provider {
  simple6 = {
    source  = "registry.terraform.io/hashicorp/simple6"
    version = "1.0.0"
  }
}

from {
  state_store "simple6_fs" {
    provider "simple6" {}
    // workspace_dir set to v1.tfstate.d during build
  }
}
