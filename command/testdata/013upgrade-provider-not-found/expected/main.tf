provider foo {}

terraform {
  required_providers {
    bar = {
      source  = "hashicorp/bar"
      version = "1.0.0"
    }
    unknown = {
      # TF-UPGRADE-TODO
      #
      # No source detected for this provider. You must add a source address
      # in the following format:
      #
      # source = "your-registry.example.com/organization/unknown"
      #
      # For more information, see the provider source documentation:
      #
      # https://www.terraform.io/docs/language/providers/requirements.html
      version = "~> 2.0.0"
    }
    foo = {
      source = "hashicorp/foo"
    }
  }
}
