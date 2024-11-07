
variable "input" {
    type = string
    default = "Hello, world!"
}

data "null_data_source" "values" {
    inputs = {
        data = var.input
    }
}

output "input" {
    value = data.null_data_source.values.outputs["data"]
}
