from {
  backend "local" {
    workspace_dir = "old-location"                  # Will be moved during the migration
    path = "old-location/default/terraform.tfstate" # Will be moved during the migration
  }
}
