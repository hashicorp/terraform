provider "test" {
  region = "somewhere"
}

variable "test_var" {
  default = "bar"
}

action "test_action" "hello" {
  count = 3
  config {
    attr = "Hello, World #${count.index}!"
  }
}

resource "test_instance" "test" {
  ami = var.test_var

  lifecycle {
    action_trigger {
      events  = [before_create]
      actions = [action.test_action.hello]
    }
  }
}

output "test" {
  value = var.test_var
}
