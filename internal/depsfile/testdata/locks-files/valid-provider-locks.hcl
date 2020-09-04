
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

  hashes {
    amigaos_m68k = [
      "placeholder-hash-1",
    ]
    tos_m68k = [
      "placeholder-hash-2",
      "placeholder-hash-3",
    ]
  }
}
