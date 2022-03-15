# This is a simple configuration with Terraform Cloud mode minimally
# activated, but it's suitable only for testing things that we can exercise
# without actually accessing Terraform Cloud, such as checking of invalid
# command-line options to "terraform init".

terraform {
  cloud {
    organization = "PLACEHOLDER"
    workspaces {
        name = "PLACEHOLDER"
    }
  }
}
