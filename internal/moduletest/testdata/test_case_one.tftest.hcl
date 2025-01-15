variables {
  input = "default"
}

# test_run_one runs a partial plan
# run "test_run_one" {
#   command = plan

#   plan_options {
#     target = [
#       foo_resource.a
#     ]
#   }

#   assert {
#     condition = foo_resource.a.value == "default"
#     error_message = "invalid value"
#   }
# }

# # test_run_two does a complete apply operation
# run "test_run_two" {
#   variables {
#     input = "custom"
#   }

#   assert {
#     condition = foo_resource.a.value == run.test_run_one.name
#     error_message = "invalid value"
#   }
# }


run "test1" {
  command = plan

  assert {
    condition = foo_resource.a.value == "default"
    error_message = "description not matching"
  }
}

run "test2" {
  command = plan

  assert {
    condition = foo_resource.a.value == run.test1.name
    error_message = "description not matching"
  }
}

run "test3" {
  command = plan

  assert {
    condition = foo_resource.a.value == run.test2.name
    error_message = "description not matching"
  }
}

run "test4" {
  command = plan

  assert {
    condition = foo_resource.a.value == "default"
    error_message = "description not matching"
  }
}

// 1 and 4 should be able to run in parallel