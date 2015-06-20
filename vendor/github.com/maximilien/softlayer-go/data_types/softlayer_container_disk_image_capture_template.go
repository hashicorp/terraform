package data_types

type SoftLayer_Container_Disk_Image_Capture_Template struct {
	Description string                                                   `json:"description"`
	Name        string                                                   `json:"name"`
	Summary     string                                                   `json:"summary"`
	Volumes     []SoftLayer_Container_Disk_Image_Capture_Template_Volume `json:"volumes"`
}

type SoftLayer_Container_Disk_Image_Capture_Template_Volume struct {
	Name       string `json:"name"`
	Partitions []SoftLayer_Container_Disk_Image_Capture_Template_Volume_Partition
}

type SoftLayer_Container_Disk_Image_Capture_Template_Volume_Partition struct {
	Name string `json:"name"`
}
