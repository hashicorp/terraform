state_store_provider {
  test = {
    source = "hashicorp/test"
    version = "1.0.0"
  }
}

migrate_from_backend "s3" {
  bucket = "foobar"
}
