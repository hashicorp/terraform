terraform {
  backend "inmem" {
    test_nested_attr_single = {
      child = "" // to be overwritten in test
    }
  }
}
