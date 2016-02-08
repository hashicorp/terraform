package mysql

type Field struct {
	Catalog  string
	Db       string
	Table    string
	OrgTable string
	Name     string
	OrgName  string
	DispLen  uint32
	//  Charset  uint16
	Flags uint16
	Type  byte
	Scale byte
}
