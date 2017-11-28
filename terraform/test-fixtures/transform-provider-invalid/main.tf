provider "aws" {
}

module "mod" {
  source = "./mod"
  
  # aws.foo doesn't exist, and should report an error
  providers = {
    "aws" = "aws.foo"
  }
}
