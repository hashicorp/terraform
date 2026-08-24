resource "test_instance" "b" {}

import {
  to = test_instance.b
  id = "test-b"
}
