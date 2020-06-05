provider "aws" {
    root = 1
}

provider "aws" {
    value = "eu"
    alias = "eu"
}

module "child" {
    source = "./child"
    providers = {
      "aws.eu" = "aws.eu"
    }
}
