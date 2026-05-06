state_store_provider { 
  test = { 
    source = "hashicorp/test" 
    version = "1.0.0"
    foobar = "this shouldn't be here" 
  }
}

migrate_from_state_store "test_store" { 
  provider "test" { 
    provider_attr = "foobar" 
  } 
  store_attr = "foobar" 
}