package models

type RouterGroups []RouterGroup

type RouterGroup struct {
	GUID string `json:"guid"`
	Name string `json:"name"`
	Type string `json:"type"`
}
