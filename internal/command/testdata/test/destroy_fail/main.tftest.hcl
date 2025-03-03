
run "setup" {
  module {
    source = "./setup"
  }
}

run "single" {}

run "double" {
  state_key = "double"
}
