provider "test" {
  resource_prefix_aaa = "foo" // this is decoded during the test runtime, and we should
  // catch that the provider configuration is invalid
}

run "test-1" {
  command = plan
}
