
variable "input" {
    type = string
    default = "Hello, world!"
}

resource "time_sleep" "wait_5_seconds" {
    create_duration = "5s"
}

resource "tfcoremock_simple_resource" "resource" {
    string = var.input

    depends_on = [ time_sleep.wait_5_seconds ]
}
