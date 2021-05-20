locals {
    l = data.null_data_source.d.id
}

data "null_data_source" "d" {
}

resource "null_resource" "a" {
    count = local.l == "NONE" ? 1 : 0
}

provider "test" {
  foo = data.null_data_source.d.id
}
