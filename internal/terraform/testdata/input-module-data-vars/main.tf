data "null_data_source" "bar" {
  foo = ["a", "b"]
}

module "child" {
  source = "./child"
  in = "${data.null_data_source.bar.foo[1]}"
}
