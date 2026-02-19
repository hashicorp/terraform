locals {
  test1 = local.test2
  test2 = local.test1
}

resource "test_instance" "foo" {
    ami = resource.test_instance.bar.ami
}

resource "test_instance" "bar" {
    ami = resource.test_instance.foo.ami
}
