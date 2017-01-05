package net

import (
	"encoding/json"
	"reflect"
)

func NewPaginatedResources(exampleResource interface{}) PaginatedResources {
	return PaginatedResources{
		resourceType: reflect.TypeOf(exampleResource),
	}
}

type PaginatedResources struct {
	NextURL        string          `json:"next_url"`
	ResourcesBytes json.RawMessage `json:"resources"`
	resourceType   reflect.Type
}

func (pr PaginatedResources) Resources() ([]interface{}, error) {
	slicePtr := reflect.New(reflect.SliceOf(pr.resourceType))
	err := json.Unmarshal([]byte(pr.ResourcesBytes), slicePtr.Interface())
	slice := reflect.Indirect(slicePtr)

	contents := make([]interface{}, 0, slice.Len())
	for i := 0; i < slice.Len(); i++ {
		contents = append(contents, slice.Index(i).Interface())
	}
	return contents, err
}
