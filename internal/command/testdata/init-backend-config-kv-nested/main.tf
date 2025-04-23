terraform {
  backend "inmem" {
    test_nesting_single = {
      child = "" // to be overwritten in test
    }
  }
}
