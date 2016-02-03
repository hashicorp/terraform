package azure

type DeleteResourceGroup struct {
	Name string `json:"-"`
}

func (s DeleteResourceGroup) ApiInfo() ApiInfo {
	return ApiInfo{
		ApiVersion:  resourceGroupAPIVersion,
		Method:      "DELETE",
		URLPathFunc: resourceGroupDefaultURLFunc(s.Name),
		ResponseTypeFunc: func() interface{} {
			return nil
		},
	}
}
