data "null_data_source" "foo" {
       count = 1
}


output "output" {
  value = data.null_data_source.foo[0].output
}

