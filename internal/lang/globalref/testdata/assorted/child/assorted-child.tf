variable "a" {
}

resource "test_thing" "foo" {
  string = var.a
}

output "a" {
  value = {
    a   = var.a
    foo = test_thing.foo
  }
}
