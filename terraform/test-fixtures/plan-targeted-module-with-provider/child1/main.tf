variable "key" {}
provider "null" { key = "${var.key}" }

resource "null_resource" "foo" {}
