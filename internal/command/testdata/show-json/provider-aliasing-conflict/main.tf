provider "test" {
  region = "somewhere"
}

resource "test_instance" "test" {
  ami = "foo"
}

module "child" {
  source = "./child"
}
