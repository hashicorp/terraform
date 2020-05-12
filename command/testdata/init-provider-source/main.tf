terraform {
  required_providers {
    test = {
      # Terraform >= v0.12.25 prints a warning that "source" is ignored
      source = "hashicorp/test"
    }
  }
}
