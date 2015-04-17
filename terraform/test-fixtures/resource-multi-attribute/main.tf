resource "resource" "name" {
  foo {
    bar1 = "baz1"
    bar2 = "baz12"
  }

  foo {
    bar1 = "baz2"
    bar2 = "baz22"
  }

  foo {
    bar1 = "baz3"
    bar2 = "baz32"
  }

  foo {
    bar1 = "baz4"
    bar2 = "baz42"
  }
}
