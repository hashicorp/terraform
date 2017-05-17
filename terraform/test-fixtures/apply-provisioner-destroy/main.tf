resource "aws_instance" "foo" {
    foo = "bar"

    provisioner "shell" {
        foo = "create"
    }

    provisioner "shell" {
        foo  = "destroy"
        when = "destroy"
    }
}
