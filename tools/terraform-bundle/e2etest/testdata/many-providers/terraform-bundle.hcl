terraform {
  version = "0.10.1"
}

providers {
  aws = ["~> 0.1"]
  kubernetes = ["0.1.0", "0.1.1", "0.1.2"]
  null = ["0.1.0"]
}