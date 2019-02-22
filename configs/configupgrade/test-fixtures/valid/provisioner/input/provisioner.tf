resource "test_instance" "foo" {
  provisioner "test" {
    commands = "${list("a", "b", "c")}"

    when       = "create"
    on_failure = "fail"
  }
}
