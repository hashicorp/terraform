provider "vsphere"
{
	user = "AniketS@ad2lab.com"
	password = "gsLab123"
	vsphere_server = "192.168.32.254"
	allow_unverified_ssl = "true"
}

resource "vsphere_virtual_machine" "temp"{
	name   = "terraform-web"
  	vcpu   = 2
  	memory = 4096

  network_interface {
    label = "vLan10vm"
  }

  disk {
    template = "Templates/Linux/RHEL7.1-ext4_Template"
  }
}
