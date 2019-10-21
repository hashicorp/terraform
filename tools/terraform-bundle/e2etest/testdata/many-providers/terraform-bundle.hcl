terraform {
  version = "0.12.0"
}

providers {
  aws        = ["~> 2.26.0"]
  kubernetes = ["1.8.0", "1.8.1", "1.9.0"]
  null       = ["2.1.0"]
}
