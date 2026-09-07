state_store_provider {
  test = {
    source = "hashicorp/test"
    version = "1.2.3"
  }
}

from {
  state_store "test_store" {
    value = "source-pss.tfstate"

    provider "test" {}
  }
}
