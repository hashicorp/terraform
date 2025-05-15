
stack "a" {
  source = "./child"

  inputs = {
    a = stack.a.a
  }
}
