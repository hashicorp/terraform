resource "test_instance" "foo" {
  image = "${sha256(file("foo.txt"))}"
}
