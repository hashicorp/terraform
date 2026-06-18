resource "test_instance" "foo" {
  ami = "bar"

  network_interface {
    device_index = 0
    description  = "Main network interface"
  }
}

data "test_data_source" "a" {
  id = "zzzzz"
}

module "mod0" {
  source = "./mod0"
}

module "mod1" {
  source = "./mod1"
}

module "mod2" {
  source = "./mod2"
}

module "mod3" {
  source = "./mod3"
}

module "mod4" {
  source = "./mod4"
}

module "mod5" {
  source = "./mod5"
}

module "mod6" {
  source = "./mod6"
}

module "mod7" {
  source = "./mod7"
}
