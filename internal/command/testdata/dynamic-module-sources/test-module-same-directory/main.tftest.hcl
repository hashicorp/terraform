run "dynamic_source" {
  command = plan

  module {
    source = "./."
  }

  assert {
    condition     = module.const_var_source.message == "hello"
    error_message = "unexpected message from dynamically-sourced module"
  }
}
