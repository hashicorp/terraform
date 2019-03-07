package resource

func GetTZData(name string) ([]byte, bool) {
	data, ok := files["zoneinfo/"+name]
	return data, ok
}
