state_store_provider { 
  test = { 
    source = "hashicorp/test" 
    version = "1.0.0" 
  }
  foobar = { 
    source = "hashicorp/foobar" 
    version = "1.0.0" 
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