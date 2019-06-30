variable "im_a_string" {
    type = "string"
}

variable "service_region_ami" {
    type = "map"
    default = {
        us-east-1 = "ami-e4c9db8e"
    }
}

resource "null_resource" "noop" {}
