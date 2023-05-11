
provider "local" {}

import {
  provider = local
  id = "foo/bar"
  to = local_file.foo_bar
}

resource "local_file" "foo_bar" {}
