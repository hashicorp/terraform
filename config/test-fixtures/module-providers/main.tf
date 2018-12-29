module "child" {
  source = "./child"
  version = "0.1.2"
  providers = {
    "aws" = "aws.foo"
  }
}
