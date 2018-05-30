resource "aws_instance" "foo" {
    foo = "bar"

    provisioner "shell" {
        command = "create"
    }

    provisioner "shell" {
        command  = "destroy"
        when = "destroy"
    }
}
