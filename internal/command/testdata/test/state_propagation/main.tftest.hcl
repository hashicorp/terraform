# Our test will run this in verbose mode and we should see the plan output for
# the second run block showing the resource being updated as the state should
# be propagated from the first one to the second one.
#
# We also interweave alternate modules to test the handling of multiple states
# within the file.

run "initial_apply_example" {
  module {
    source = "./example"
  }

  variables {
    input = "start"
  }
}

run "initial_apply" {
  variables {
    input = "start"
  }
}

run "plan_second_example" {
  command = plan

  module {
    source = "./second_example"
  }

  variables {
    input = "start"
  }
}

run "plan_update" {
  command = plan

  variables {
    input = "update"
  }
}

run "plan_update_example" {
  command = plan

  module {
    source = "./example"
  }

  variables {
    input = "update"
  }
}
