variable "users" {
    default = {
        one = "onepw"
        two = "twopw"
    }
}

provider "test" {
    url = "example.com"
    
    dynamic "auth" {
        for_each = var.users
        content {
            user     = auth.key
            password = auth.value
        }
    }
}

resource "test_instance" "test" {}