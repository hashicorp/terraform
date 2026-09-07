state_store_provider { 
  foobar = { 
    source = "hashicorp/foobar" 
    version = "1.0.0" 
  }
}

# The state store below references a different provider to the definition above

from {
  state_store  "test_store" { 
    provider "test" { 
      provider_attr = "foobar" 
    } 
    store_attr = "foobar" 
    }
}