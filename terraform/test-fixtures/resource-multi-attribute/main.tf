resource "resource" "name" {
  foo {
    bar = "baz1"
  }

  foo {
    bar = "baz2"
  }

  foo {
    bar = "baz3"
  }

  foo {
    bar = "baz4"
  }
}
