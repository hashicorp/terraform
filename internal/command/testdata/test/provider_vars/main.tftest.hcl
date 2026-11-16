
variables {
  resource_directory = "my-resource-dir"
}

provider "test" {
  resource_prefix = var.resource_directory
}

run "test" {}
