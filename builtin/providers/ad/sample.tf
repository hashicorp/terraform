provider "ad" {
    domain = "exampledomain.com"
    user = "user"
    password = "password"
    ip = "domain_ip"
}
    
resource "ad_resourceComputer" "foo"{
	domain = "exampledomain.com"
	computer_name = "terraformSample"
}
