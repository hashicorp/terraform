package dvs

type dvs_map_host_dvs struct {
	hostName   string
	switchName string
	nicName    []string
}

type dvs_map_vm_dvpg struct {
	vm        string
	nicLabel  string
	portgroup string
}
