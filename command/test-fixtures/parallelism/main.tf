resource "test_instance" "foo1" {
	ami = "bar"

	// shell has been configured to sleep for one second
	provisioner "shell" {}
}

resource "test_instance" "foo2" {
	ami = "bar"

	// shell has been configured to sleep for one second
	provisioner "shell" {}
}
