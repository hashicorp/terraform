
run "one" {
  state_key = ""
  module {
    source = "./setup"
  }
}

run "two" {
  command = plan
}
