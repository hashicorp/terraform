package profile

type ClientProfile struct {
	HttpProfile *HttpProfile
	// Valid choices: HmacSHA1, HmacSHA256, TC3-HMAC-SHA256.
	// Default value is TC3-HMAC-SHA256.
	SignMethod      string
	UnsignedPayload bool
	// Valid choices: zh-CN, en-US.
	// Default value is zh-CN.
	Language string
}

func NewClientProfile() *ClientProfile {
	return &ClientProfile{
		HttpProfile:     NewHttpProfile(),
		SignMethod:      "TC3-HMAC-SHA256",
		UnsignedPayload: false,
		Language:        "zh-CN",
	}
}
