package data_types

type SoftLayer_Security_Certificate struct {
	Id                      int    `json:"id"`
	Certificate             string `json:"certificate"`
	IntermediateCertificate string `json:"intermediateCertificate"`
	PrivateKey              string `json:"privateKey"`
	CommonName              string `json:"commonName"`
	OrganizationName        string `json:"organizationName"`
	ValidityBegin           string `json:"validityBegin"`
	ValidityDays            int    `json:"validityDays"`
	ValidityEnd             string `json:"validityEnd"`
	KeySize                 int    `json:"keySize"`
	CreateDate              string `json:"createDate"`
	ModifyDate              string `json:"modifyDate"`
}

type SoftLayer_Security_Certificate_Parameters struct {
	Parameters []SoftLayer_Security_Certificate_Template `json:"parameters"`
}

type SoftLayer_Security_Certificate_Template struct {
	Certificate             string `json:"certificate"`
	IntermediateCertificate string `json:"intermediateCertificate"`
	PrivateKey              string `json:"privateKey"`
}
