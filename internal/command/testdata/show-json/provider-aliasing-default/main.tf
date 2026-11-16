provider "test" {
  region = "somewhere"
}

provider "test" {
  alias  = "backup"
  region = "elsewhere"
}

resource "test_instance" "test" {
  ami = "foo"
}

module "child" {
  source = "./child"
  providers = {
    test = test.backup
  }
}
