/**
External interface to the DVS module that exposes the plug-in
to Terraform
**/
package dvs

import "github.com/hashicorp/terraform/helper/schema"

// ResourceVSphereDVS exposes the DVS resource
func ResourceVSphereDVS() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereDVSCreate,
		Read:   resourceVSphereDVSRead,
		Update: resourceVSphereDVSUpdate,
		Delete: resourceVSphereDVSDelete,
		Schema: resourceVSphereDVSSchema(),
	}
}

// ResourceVSphereDVPG exposes the DVPG resource
func ResourceVSphereDVPG() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereDVPGCreate,
		Read:   resourceVSphereDVPGRead,
		Update: resourceVSphereDVPGUpdate,
		Delete: resourceVSphereDVPGDelete,
		Schema: resourceVSphereDVPGSchema(),
	}
}

// ResourceVSphereMapHostDVS exposes the MapHostDVS resource (untested)
func ResourceVSphereMapHostDVS() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereMapHostDVSCreate,
		Read:   resourceVSphereMapHostDVSRead,
		Delete: resourceVSphereMapHostDVSDelete,
		Schema: resourceVSphereMapHostDVSSchema(),
	}
}

// ResourceVSphereMapVMDVPG exposes the MapVMDVPG resource
func ResourceVSphereMapVMDVPG() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereMapVMDVPGCreate,
		Read:   resourceVSphereMapVMDVPGRead,
		// Update: resourceVSphereMapVMDVPGUpdate, // not needed
		Delete: resourceVSphereMapVMDVPGDelete,
		Schema: resourceVSphereMapVMDVPGSchema(),
	}
}
