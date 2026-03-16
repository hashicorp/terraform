
required_providers {
  foo = {
    source = "terraform.io/builtin/foo"
  }
}

component "a" {
  source = "./module"
}

component "b" {
  source = "./module"

  inputs = {
    in = component.a.out
  }
}

output "out" {
  type  = string
  value = component.a.out
}

provider "foo" "bar" {
  config {
    in = {
      a = component.a.out
      b = component.b.out
    }
  }
}

stack "child" {
  source = "./child"

  inputs = {
    in = component.b.out
  }
}

component "c" {
  source = "./module"

  inputs = {
    # stack.child.out depends indirectly on component.b, so therefore
    # component.c should transitively depend on component.b.
    in = "${component.a.out}-${stack.child.out}"
  }
}
