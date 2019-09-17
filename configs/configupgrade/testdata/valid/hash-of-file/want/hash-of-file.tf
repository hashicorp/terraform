resource "test_instance" "foo" {
  image = filesha256("foo.txt")
}
