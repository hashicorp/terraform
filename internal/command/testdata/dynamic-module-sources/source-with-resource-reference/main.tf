resource "test_instance" "example" {}

module "example" {
  source = "./modules/${test_instance.example.id}"
}
