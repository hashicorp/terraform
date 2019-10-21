resource "test_instance" "foo" {
  network = [
    {
      cidr_block = "10.1.2.0/24"
    },
    "${var.baz}"
  ]
}
