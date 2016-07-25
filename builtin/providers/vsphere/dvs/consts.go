package dvs

import "regexp"

const maphostdvs_name_format = "vSphere::MapHostDVS::%s::%s::%s"

var re_maphostdvs *regexp.Regexp

const mapvmdvpg_name_format = "vSphere::MapVMDVPG::%s::%s::%s::%s"

var re_mapvmdvpg *regexp.Regexp

const dvpg_name_format = "vSphere::DVPG::%s::%s::%s"

var re_dvpg *regexp.Regexp

const dvs_name_format = "vSphere::DVS::%s::%s"

var re_dvs *regexp.Regexp

func init() {
	re_dvs = regexp.MustCompile(`vSphere::DVS::(.*?)::(.*)$`)
	re_dvpg = regexp.MustCompile(`vSphere::DVPG::(.*?)::(.*?)::(.*)$`)
	re_maphostdvs = regexp.MustCompile(`vSphere::MapHostDVS::(.*?)::(.*?)::(.*)$`)
	re_mapvmdvpg = regexp.MustCompile(`vSphere::MapVMDVPG::(.*?)::(.*?)::(.*?)::(.*)$`)

}
