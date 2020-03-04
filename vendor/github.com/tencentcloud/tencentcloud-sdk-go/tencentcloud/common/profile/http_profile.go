package profile

type HttpProfile struct {
	ReqMethod  string
	ReqTimeout int
	Endpoint   string
	Protocol   string
}

func NewHttpProfile() *HttpProfile {
	return &HttpProfile{
		ReqMethod:  "POST",
		ReqTimeout: 60,
		Endpoint:   "",
		Protocol:   "HTTPS",
	}
}
