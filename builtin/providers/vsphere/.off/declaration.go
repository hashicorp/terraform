/**
External interface to the DVS module that exposes the plug-in
to Terraform
**/
package dvs

/** disabled because untested

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
// **/
