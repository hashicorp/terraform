package search

import "github.com/jen20/riviera/azure"

type DeleteSearchService struct {
	Name              string `json:"-"`
	ResourceGroupName string `json:"-"`
}

func (s DeleteSearchService) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "DELETE",
		URLPathFunc: searchServiceDefaultURLPath(s.ResourceGroupName, s.Name),
		ResponseTypeFunc: func() interface{} {
			return nil
		},
	}
}
