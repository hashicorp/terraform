package models

type Buildpack struct {
	GUID     string
	Name     string
	Position *int
	Enabled  *bool
	Key      string
	Filename string
	Locked   *bool
}
