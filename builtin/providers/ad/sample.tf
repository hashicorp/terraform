provider "ad" {
    domain = "localscvmm.net"
    user = "scvmm_sa"
    password = "Passw0rd"
    ip = "10.136.60.97"
}
    
resource "ad_resourceComputer" "foo"{
	domain = "localscvmm.net"
	computer_name = "terraformSample"
}