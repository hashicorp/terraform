state_store_provider {
  test = {
    source = "hashicorp/test"
    version = "1.0.0"
  }
}

from {
  backend "s3" {
    bucket = "foobar"
  }
}
