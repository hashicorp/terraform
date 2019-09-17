resource "test_instance" "foo" {
  type  = "z1.weedy"
  image = "image-abcd"
  tags {
    name = "boop"
  }
}
