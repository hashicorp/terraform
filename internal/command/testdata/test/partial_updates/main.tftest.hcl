
run "first" {
  plan_options {
    target = [
      test_resource.resource,
    ]
  }
}

run "second" {}