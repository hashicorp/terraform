provider "scvmm" {
        server_ip = "<scvmm_server_ip>"
        port = <scvmm_server_winrm_port>
        user_name = "<scvmm_server_user>"
        user_password = "<scvmm_server_password>"
}

resource "scvmm_vm" "demoCreateVM" {
        timeout = "10000"
        vmm_server = "WIN-2F929KU8HIU"
        vm_name = "Test_VM_demo01"
        template_name = "TestVMTemplate"
        cloud_name = "GSL Cloud"
}

resource "scvmm_virtual_disk" "demoDisk" {
        timeout = 10000
        vmm_server = "WIN-2F929KU8HIU"
        vm_name = "${scvmm_vm.CreateVM.vm_name}"
        virtual_disk_name = "demo_disk"
        virtual_disk_size = 10000
}

resource "scvmm_start_vm" "demoStart" {
        timeout=1000
        vmm_server= "WIN-2F929KU8HIU"
        vm_name = "${scvmm_vm.CreateVM.vm_name}"
}

resource "scvmm_stop_vm" "demoStop" {
        timeout=1000
        vmm_server= "WIN-2F929KU8HIU"
        vm_name = "${scvmm_vm.CreateVM.vm_name}"

}
resource "scvmm_checkpoint" "demoCheck" {
        timeout=1000
        vmm_server="WIN-2F929KU8HIU"
        vm_name="${scvmm_vm.CreateVM.vm_name}"
        checkpoint_name="demo_checkpoint"
}

resource "scvmm_revert_checkpoint" "demoRevert" {
	     timeout=1000
        vmm_server="WIN-2F929KU8HIU"
        vm_name="${scvmm_vm.CreateVM.vm_name}"
        checkpoint_name="demo_checkpoint"
}