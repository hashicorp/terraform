package compute

const (
	ImageListDescription   = "Image List"
	ImageListContainerPath = "/imagelist/"
	ImageListResourcePath  = "/imagelist"
)

// ImageListClient is a client for the Image List functions of the Compute API.
type ImageListClient struct {
	ResourceClient
}

// ImageList obtains an ImageListClient which can be used to access to the
// Image List functions of the Compute API
func (c *Client) ImageList() *ImageListClient {
	return &ImageListClient{
		ResourceClient: ResourceClient{
			Client:              c,
			ResourceDescription: ImageListDescription,
			ContainerPath:       ImageListContainerPath,
			ResourceRootPath:    ImageListResourcePath,
		}}
}

type ImageListEntry struct {
	// User-defined parameters, in JSON format, that can be passed to an instance of this machine image when it is launched.
	Attributes map[string]interface{} `json:"attributes"`

	// Name of the Image List.
	ImageList string `json:"imagelist"`

	// A list of machine images.
	MachineImages []string `json:"machineimages"`

	// Uniform Resource Identifier.
	URI string `json:"uri"`

	// Version number of these Machine Images in the Image List.
	Version int `json:"version"`
}

// ImageList describes an existing Image List.
type ImageList struct {
	// The image list entry to be used, by default, when launching instances using this image list
	Default int `json:"default"`

	// A description of this image list.
	Description string `json:"description"`

	// Each machine image in an image list is identified by an image list entry.
	Entries []ImageListEntry `json:"entries"`

	// The name of the Image List
	Name string `json:"name"`

	// Uniform Resource Identifier
	URI string `json:"uri"`
}

// CreateImageListInput defines an Image List to be created.
type CreateImageListInput struct {
	// The image list entry to be used, by default, when launching instances using this image list.
	// If you don't specify this value, it is set to 1.
	// Optional
	Default int `json:"default"`

	// A description of this image list.
	// Required
	Description string `json:"description"`

	// The name of the Image List
	// Object names can contain only alphanumeric characters, hyphens, underscores, and periods. Object names are case-sensitive.
	// Required
	Name string `json:"name"`
}

// CreateImageList creates a new Image List with the given name, key and enabled flag.
func (c *ImageListClient) CreateImageList(createInput *CreateImageListInput) (*ImageList, error) {
	var imageList ImageList
	createInput.Name = c.getQualifiedName(createInput.Name)
	if err := c.createResource(&createInput, &imageList); err != nil {
		return nil, err
	}

	return c.success(&imageList)
}

// DeleteKeyInput describes the image list to delete
type DeleteImageListInput struct {
	// The name of the Image List
	Name string `json:name`
}

// DeleteImageList deletes the Image List with the given name.
func (c *ImageListClient) DeleteImageList(deleteInput *DeleteImageListInput) error {
	deleteInput.Name = c.getQualifiedName(deleteInput.Name)
	return c.deleteResource(deleteInput.Name)
}

// GetImageListInput describes the image list to get
type GetImageListInput struct {
	// The name of the Image List
	Name string `json:name`
}

// GetImageList retrieves the Image List with the given name.
func (c *ImageListClient) GetImageList(getInput *GetImageListInput) (*ImageList, error) {
	getInput.Name = c.getQualifiedName(getInput.Name)

	var imageList ImageList
	if err := c.getResource(getInput.Name, &imageList); err != nil {
		return nil, err
	}

	return c.success(&imageList)
}

// UpdateImageListInput defines an Image List to be updated
type UpdateImageListInput struct {
	// The image list entry to be used, by default, when launching instances using this image list.
	// If you don't specify this value, it is set to 1.
	// Optional
	Default int `json:"default"`

	// A description of this image list.
	// Required
	Description string `json:"description"`

	// The name of the Image List
	// Object names can contain only alphanumeric characters, hyphens, underscores, and periods. Object names are case-sensitive.
	// Required
	Name string `json:"name"`
}

// UpdateImageList updates the key and enabled flag of the Image List with the given name.
func (c *ImageListClient) UpdateImageList(updateInput *UpdateImageListInput) (*ImageList, error) {
	var imageList ImageList
	updateInput.Name = c.getQualifiedName(updateInput.Name)
	if err := c.updateResource(updateInput.Name, updateInput, &imageList); err != nil {
		return nil, err
	}
	return c.success(&imageList)
}

func (c *ImageListClient) success(imageList *ImageList) (*ImageList, error) {
	c.unqualify(&imageList.Name)

	for _, v := range imageList.Entries {
		v.MachineImages = c.getUnqualifiedList(v.MachineImages)
	}

	return imageList, nil
}
