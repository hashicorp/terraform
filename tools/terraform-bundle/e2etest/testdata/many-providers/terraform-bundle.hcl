terraform {
  version = "0.13.0"
}

providers {
  aws = {
    versions = ["~> 2.26.0"]
  }

  kubernetes = {
    versions = ["1.8.0", "1.8.1", "1.9.0"]
  }

  null = {
    versions = ["2.1.0"]
  }
}
