data "null_data_source" "foo" {
  foo = "yes"
}

data "null_data_source" "bar" {
  bar = "${data.null_data_source.foo.foo}"
}
