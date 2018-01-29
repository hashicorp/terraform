variable "value" {
}

resource "aws_instance" "bar" {
    provisioner "shell" {
        command  = "${var.value}"
        when = "destroy"
    }
}

