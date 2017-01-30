package profitbricks

// slash returns "/<str>" great for making url paths
func slash(str string) string {
	return "/" + str
}

// dc_col_path	returns the string "/datacenters"
func dc_col_path() string {
	return slash("datacenters")
}

// dc_path returns the string "/datacenters/<dcid>"
func dc_path(dcid string) string {
	return dc_col_path() + slash(dcid)
}

// image_col_path returns the string" /images"
func image_col_path() string {
	return slash("images")
}

// image_path returns the string"/images/<imageid>"
func image_path(imageid string) string {
	return image_col_path() + slash(imageid)
}

// ipblock_col_path returns the string "/ipblocks"
func ipblock_col_path() string {
	return slash("ipblocks")
}

//  ipblock_path returns the string "/ipblocks/<ipblockid>"
func ipblock_path(ipblockid string) string {
	return ipblock_col_path() + slash(ipblockid)
}

// location_col_path returns the string  "/locations"
func location_col_path() string {
	return slash("locations")
}

// location_path returns the string   "/locations/<locid>"
func location_path(locid string) string {
	return location_col_path() + slash(locid)
}

// snapshot_col_path returns the string "/snapshots"
func snapshot_col_path() string {
	return slash("snapshots")
}

// lan_col_path returns the string "/datacenters/<dcid>/lans"
func lan_col_path(dcid string) string {
	return dc_path(dcid) + slash("lans")
}

// lan_path returns the string	"/datacenters/<dcid>/lans/<lanid>"
func lan_path(dcid, lanid string) string {
	return lan_col_path(dcid) + slash(lanid)
}

//  lbal_col_path returns the string "/loadbalancers"
func lbal_col_path(dcid string) string {
	return dc_path(dcid) + slash("loadbalancers")
}

// lbalpath returns the string "/loadbalancers/<lbalid>"
func lbal_path(dcid, lbalid string) string {
	return lbal_col_path(dcid) + slash(lbalid)
}

// server_col_path returns the string	"/datacenters/<dcid>/servers"
func server_col_path(dcid string) string {
	return dc_path(dcid) + slash("servers")
}

// server_path returns the string   "/datacenters/<dcid>/servers/<srvid>"
func server_path(dcid, srvid string) string {
	return server_col_path(dcid) + slash(srvid)
}

// server_cmd_path returns the string   "/datacenters/<dcid>/servers/<srvid>/<cmd>"
func server_command_path(dcid, srvid, cmd string) string {
	return server_path(dcid, srvid) + slash(cmd)
}

// volume_col_path returns the string "/volumes"
func volume_col_path(dcid string) string {
	return dc_path(dcid) + slash("volumes")
}

// volume_path returns the string "/volumes/<volid>"
func volume_path(dcid, volid string) string {
	return volume_col_path(dcid) + slash(volid)
}

//  balnic_col_path returns the string "/loadbalancers/<lbalid>/balancednics"
func balnic_col_path(dcid, lbalid string) string {
	return lbal_path(dcid, lbalid) + slash("balancednics")
}

//  balnic_path returns the string "/loadbalancers/<lbalid>/balancednics<balnicid>"
func balnic_path(dcid, lbalid, balnicid string) string {
	return balnic_col_path(dcid, lbalid) + slash(balnicid)
}

// server_cdrom_col_path returns the string   "/datacenters/<dcid>/servers/<srvid>/cdroms"
func server_cdrom_col_path(dcid, srvid string) string {
	return server_path(dcid, srvid) + slash("cdroms")
}

// server_cdrom_path returns the string   "/datacenters/<dcid>/servers/<srvid>/cdroms/<cdid>"
func server_cdrom_path(dcid, srvid, cdid string) string {
	return server_cdrom_col_path(dcid, srvid) + slash(cdid)
}

// server_volume_col_path returns the string   "/datacenters/<dcid>/servers/<srvid>/volumes"
func server_volume_col_path(dcid, srvid string) string {
	return server_path(dcid, srvid) + slash("volumes")
}

// server_volume_path returns the string   "/datacenters/<dcid>/servers/<srvid>/volumes/<volid>"
func server_volume_path(dcid, srvid, volid string) string {
	return server_volume_col_path(dcid, srvid) + slash(volid)
}

// nic_path returns the string   "/datacenters/<dcid>/servers/<srvid>/nics"
func nic_col_path(dcid, srvid string) string {
	return server_path(dcid, srvid) + slash("nics")
}

// nic_path returns the string   "/datacenters/<dcid>/servers/<srvid>/nics/<nicid>"
func nic_path(dcid, srvid, nicid string) string {
	return nic_col_path(dcid, srvid) + slash(nicid)
}

// fwrule_col_path returns the string   "/datacenters/<dcid>/servers/<srvid>/nics/<nicid>/firewallrules"
func fwrule_col_path(dcid, srvid, nicid string) string {
	return nic_path(dcid, srvid, nicid) + slash("firewallrules")
}

// fwrule_path returns the string
//  "/datacenters/<dcid>/servers/<srvid>/nics/<nicid>/firewallrules/<fwruleid>"
func fwrule_path(dcid, srvid, nicid, fwruleid string) string {
	return fwrule_col_path(dcid, srvid, nicid) + slash(fwruleid)
}
