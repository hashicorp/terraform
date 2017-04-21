resource "aws_instance" "foo" {
    foo = "bar"

    provisioner "shell" {
        foo  = "one"
        when = "destroy"
        on_failure = "continue"
    }

    provisioner "shell" {
        foo  = "two"
        when = "destroy"
    }
}
