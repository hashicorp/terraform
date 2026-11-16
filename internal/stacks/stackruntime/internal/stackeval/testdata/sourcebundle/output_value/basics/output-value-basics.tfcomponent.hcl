output "root" {
  type  = string
  value = _test_only_global.root_output
}

output "child" {
  type  = string
  value = stack.child.foo
}

stack "child" {
  source = "./child"
}
