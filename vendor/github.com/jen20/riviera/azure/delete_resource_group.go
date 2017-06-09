package azure

type DeleteResourceGroup struct {
	Name string `json:"-"`
}

func (s DeleteResourceGroup) APIInfo() APIInfo {
	return APIInfo{
		APIVersion:  resourceGroupAPIVersion,
		Method:      "DELETE",
		URLPathFunc: resourceGroupDefaultURLFunc(s.Name),
		ResponseTypeFunc: func() interface{} {
			return nil
		},
	}
}
