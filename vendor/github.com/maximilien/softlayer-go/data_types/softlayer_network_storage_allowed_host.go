package data_types

type SoftLayer_Network_Storage_Allowed_Host struct {
	CredentialId      int    `json:"credentialId"`
	Id                int    `json:"id"`
	Name              string `json:"name"`
	ResourceTableId   int    `json:"resourceTabledId"`
	ResourceTableName string `jsob:"resourceTableName"`
}
