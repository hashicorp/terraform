provider "test" {

}

# No resource block present, so import fails

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
