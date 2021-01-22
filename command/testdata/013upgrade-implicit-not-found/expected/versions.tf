terraform {
  required_providers {
    something = {
      # TF-UPGRADE-TODO
      #
      # No source detected for this provider. You must add a source address
      # in the following format:
      #
      # source = "your-registry.example.com/organization/something"
      #
      # For more information, see the provider source documentation:
      #
      # https://www.terraform.io/docs/language/providers/requirements.html
    }
  }
  required_version = ">= 0.13"
}
