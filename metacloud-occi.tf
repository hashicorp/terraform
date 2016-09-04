resource "occi_virtual_machine" "vm_small" {
	image_template = "http://occi.carach5.ics.muni.cz/occi/infrastructure/os_tpl#uuid_egi_centos_7_fedcloud_warg_149"
	resource_template = "http://fedcloud.egi.eu/occi/compute/flavour/1.0#small"
	endpoint = "https://carach5.ics.muni.cz:11443"
	name = "occi.core.title=test_vm_small"
	x509 = "/tmp/x509up_u1000"
}

output "virtual_machine_id" {
	value = "${occi_virtual_machine.vm_small.vm_id}"
}