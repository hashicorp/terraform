provider "null" {
  value = ""
}

module "mod" {
  source = "./mod"
}

provider "test" {
  value = module.mod.output
}

resource "test_instance" "bar" {
}

