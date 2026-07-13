resource "random_pet" "root" {
    lifecycle {
      destroy = false
    }
}

module "child" {
    source = "./child"
}