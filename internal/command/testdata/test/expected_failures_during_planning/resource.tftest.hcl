
run "resource_failure" {

  variables {
    input = "acd"
  }

  # While we do expect test_resource.resource to fail, we are asking this run
  # block to execute an apply operation. It can't do that because our custom
  # condition fails during the planning stage as well. Our test is going to make
  # sure we add the helpful warning diagnostic explaining this.
  expect_failures = [
    test_resource.resource,
  ]

}
