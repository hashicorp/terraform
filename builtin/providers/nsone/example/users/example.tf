provider "nsone" {
}

resource "nsone_user" "example" {
    name = "Example Terraform User"
    email = "terraform-example@some.domain"
    username = "terraformexample"
}

resource "nsone_apikey" "example" {
    name = "Example Terraform API Key"
}

output "api_key" {
    value = "${nsone_apikey.example.key}"
}

