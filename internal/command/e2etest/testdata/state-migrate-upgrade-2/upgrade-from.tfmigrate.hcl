state_store_provider {
  simple6 = {
    source  = "registry.terraform.io/hashicorp/simple6"
    version = "1.0.0"
  }
}

from {
  state_store "simple6_fs" {
    provider "simple6" {}

    workspace_dir = "v1.tfstate.d"
    attr_v1 = "foobar" # Attribute name set as 'attr_v1' during build
  }
}

