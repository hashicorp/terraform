provider "test" {
  region = "somewhere"
}

provider "test" {
  alias  = "backup"
  region = "elsewhere"
}

resource "test_instance" "test" {
  ami      = "foo"
  provider = test
}

resource "test_instance" "test_backup" {
  ami      = "foo-backup"
  provider = test.backup
}

module "child" {
  source = "./child"
  providers = {
    test        = test
    test.second = test.backup
  }
}

module "sibling" {
  source = "./child"
  providers = {
    test        = test
    test.second = test
  }
}
