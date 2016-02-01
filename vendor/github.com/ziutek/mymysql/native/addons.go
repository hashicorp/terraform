package native

func NbinToNstr(nbin *[]byte) *string {
	if nbin == nil {
		return nil
	}
	str := string(*nbin)
	return &str
}

func NstrToNbin(nstr *string) *[]byte {
	if nstr == nil {
		return nil
	}
	bin := []byte(*nstr)
	return &bin
}
