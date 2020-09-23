
provider "terraform.io/test/version-only" {
  version = "1.0.0"
}

provider "terraform.io/test/version-and-constraints" {
  version = "1.2.0"
  constraints = "~> 1.2"
}

provider "terraform.io/test/all-the-things" {
  version = "3.0.10"
  constraints = ">= 3.0.2"

  hashes = [
    "test:placeholder-hash-1",
    "test:placeholder-hash-2",
    "test:placeholder-hash-3",
  ]
}
