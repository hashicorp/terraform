
module "importable_resource" {
  source = "../valid-modules/importable-resource"
}

provider "local" {}

import {
  provider = local
  id = "foo/bar"
  to = module.importable_resource.local_file.foo
}
