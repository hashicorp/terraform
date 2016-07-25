package dvs

type dvs struct {
	name         string
	folder       string
	datacenter   string
	extensionKey string
	description  string
	contact      struct {
		name  string
		infos string
	}
	switchUsagePolicy struct {
		autoPreinstallAllowed bool
		autoUpgradeAllowed    bool
		partialUpgradeAllowed bool
	}
	switchIPAddress    string
	numStandalonePorts int
}

type dvs_map_host_dvs struct {
	hostName   string
	switchName string
	nicName    []string
}

type dvs_port_range struct {
	start int
	end   int
}

type dvs_port_group struct {
	name           string
	switchId       string
	defaultVLAN    int
	vlanRanges     []dvs_port_range
	pgType         string
	description    string
	autoExpand     bool
	numPorts       int
	portNameFormat string
	policy         struct {
		allowBlockOverride         bool
		allowLivePortMoving        bool
		allowNetworkRPOverride     bool
		portConfigResetDisconnect  bool
		allowShapingOverride       bool
		allowTrafficFilterOverride bool
		allowVendorConfigOverride  bool
	}
}

type dvs_map_vm_dvpg struct {
	vm        string
	nicLabel  string
	portgroup string
}
