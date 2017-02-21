resource "aws_instance" "web" {
    provisioner "shell" {}

    provisioner "shell" {
        path = "foo"
        when = "destroy"
    }

    provisioner "shell" {
        path = "foo"
        when = "destroy"
        on_failure = "continue"
    }
}
