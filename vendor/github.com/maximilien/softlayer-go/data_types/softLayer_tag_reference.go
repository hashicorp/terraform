package data_types

type SoftLayer_Tag_Reference struct {
	EmpRecordId     *int         `json:"empRecordId"`
	Id              int          `json:"id"`
	ResourceTableId int          `json:"resourceTableId"`
	Tag             TagReference `json:"tag"`
	TagId           int          `json:"tagId"`
	TagType         TagType      `json:"tagType"`
	TagTypeId       int          `json:"tagTypeId"`
	UsrRecordId     int          `json:"usrRecordId"`
}

type TagReference struct {
	AccountId int    `json:"accountId"`
	Id        int    `json:"id"`
	Internal  int    `json:"internal"`
	Name      string `json:"name"`
}

type TagType struct {
	Description string `json:"description"`
	KeyName     string `json:"keyName"`
}
