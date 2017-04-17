resource "test" "A" {
    provisioner "foo" {
        command = "${test.A.id}"
    }
}
