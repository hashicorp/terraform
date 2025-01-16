# // To run in parallel, sequential runs must have different state keys, and not depend on each other
# // NotDepends: true
# // DiffStateKey: true

# variables {
#   foo = "foo"
# }


# run "setup" {
#   module {
#     source = "./setup"
#   }

#   variables {
#     input = "foo"
#   }

#   assert {
#     condition = output.value == var.foo
#     error_message = "bad"
#   }
# }

# // Depends on previous run, but has different state key, so would not run in parallel
# // NotDepends: false
# // DiffStateKey: true
# run "test_a" {
#   variables {
#     input = run.setup.value
#   }

#   assert {
#     condition = output.value == var.foo
#     error_message = "double bad"
#   }

#   assert {
#     condition = run.setup.value == var.foo
#     error_message = "triple bad"
#   }
# }

# // Depends on previous run, and has same state key, so would not run in parallel
# // NotDepends: false
# // DiffStateKey: false
# run "test_b" {
#   variables {
#     input = run.test_a.value
#   }

#   assert {
#     condition = output.value == var.foo
#     error_message = "double bad"
#   }

#   assert {
#     condition = run.setup.value == var.foo
#     error_message = "triple bad"
#   }
# }

# // Does not depend on previous run, and has same state key, so would not run in parallel
# // NotDepends: true
# // DiffStateKey: false
# run "test_c" {

#   variables {
#     input = "foo"
#   }

#   assert {
#     condition = output.value == var.foo
#     error_message = "double bad"
#   }

#   assert {
#     condition = run.setup.value == var.foo
#     error_message = "triple bad"
#   }
# }

# // Does not depend on previous run, and has different state key, so would run in parallel
# // NotDepends: true
# // DiffStateKey: true
# run "test_d" {

#   variables {
#     input = "foo"
#   }

#   assert {
#     condition = output.value == var.foo
#     error_message = "double bad"
#   }

#   assert {
#     condition = run.setup.value == var.foo
#     error_message = "triple bad"
#   }
# }
