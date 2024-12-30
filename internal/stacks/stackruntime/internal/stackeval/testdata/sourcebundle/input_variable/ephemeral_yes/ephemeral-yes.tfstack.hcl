
stack "child" {
  source = "./child"

  inputs = {
    a = _test_only_global.var_val
  }
}
