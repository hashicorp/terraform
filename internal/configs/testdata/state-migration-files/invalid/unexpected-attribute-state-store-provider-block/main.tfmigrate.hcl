state_store_provider { 
  test = { 
    source = "hashicorp/test" 
    version = "1.0.0"
    foobar = "this shouldn't be here" 
  }
}

from {
  state_store  "test_store" { 
    provider "test" { 
      provider_attr = "foobar" 
    } 
    store_attr = "foobar" 
  }
}