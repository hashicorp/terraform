# No state_store_provider block here as that would trigger a different error
# i.e. it is mutually exclusive with 'backend'.

from {
  backend "s3" {
    bucket = "foobar"
  }
  state_store  "test_store1" { 
    provider "test" { 
      provider_attr = "foobar" 
    } 
    store_attr = "foobar" 
  }
}