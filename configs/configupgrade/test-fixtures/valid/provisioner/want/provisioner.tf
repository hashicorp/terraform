resource "test_instance" "foo" {
  provisioner "test" {
    commands = ["a", "b", "c"]

    when       = create
    on_failure = fail
  }
}
