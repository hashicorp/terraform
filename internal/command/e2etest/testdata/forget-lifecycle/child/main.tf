resource "random_pet" "child" {
    lifecycle {
      destroy = false
    }
}