resource "example" "example" {
  lifecycle {
    create_before_destroy = "ABSOLUTELY NOT"
  }
}
