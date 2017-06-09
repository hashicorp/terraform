package data_types

type UserMetadata string
type UserMetadataArray []UserMetadata

type SoftLayer_SetUserMetadata_Parameters struct {
	Parameters []UserMetadataArray `json:"parameters"`
}
