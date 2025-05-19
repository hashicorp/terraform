variable "the_id" {
  type = string
}

import {
  to = test_resource.foo
  id = var.the_id
}

resource "test_resource" "foo" {}
