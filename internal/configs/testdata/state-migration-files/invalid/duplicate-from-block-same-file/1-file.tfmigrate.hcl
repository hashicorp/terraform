state_store_provider { 
  test = { 
    source = "hashicorp/test" 
    version = "1.0.0" 
  }
}

from {
  state_store  "test_store1" { 
    provider "test" { 
      provider_attr = "foobar" 
    } 
    store_attr = "foobar" 
  }
}

from {
  state_store  "test_store2" { 
    provider "test" { 
      provider_attr = "foobar" 
    } 
    store_attr = "foobar" 
  }
}