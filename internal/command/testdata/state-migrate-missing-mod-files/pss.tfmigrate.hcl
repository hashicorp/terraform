state_store_provider {
  test = {
    source = "hashicorp/test"
    version = "1.2.3"
  }
}

from {
  state_store "test_store" {
    path = "source.tfstate"

    provider "test" {}
  }
}
