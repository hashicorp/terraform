provider "aws" {
  foo = data.template_data_source.d.foo
}

data "template_data_source" "d" {
  foo = "bar"
}
