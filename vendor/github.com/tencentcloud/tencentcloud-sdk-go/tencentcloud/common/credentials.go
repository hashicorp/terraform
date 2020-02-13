package common

type Credential struct {
	SecretId  string
	SecretKey string
	Token     string
}

func NewCredential(secretId, secretKey string) *Credential {
	return &Credential{
		SecretId:  secretId,
		SecretKey: secretKey,
	}
}

func NewTokenCredential(secretId, secretKey, token string) *Credential {
	return &Credential{
		SecretId:  secretId,
		SecretKey: secretKey,
		Token:     token,
	}
}

func (c *Credential) GetCredentialParams() map[string]string {
	p := map[string]string{
		"SecretId": c.SecretId,
	}
	if c.Token != "" {
		p["Token"] = c.Token
	}
	return p
}

// Nowhere use them and we haven't well designed these structures and
// underlying method, which leads to the situation that it is hard to
// refactor it to interfaces.
// Hence they are removed and merged into Credential.

//type TokenCredential struct {
//	SecretId  string
//	SecretKey string
//	Token     string
//}

//func NewTokenCredential(secretId, secretKey, token string) *TokenCredential {
//	return &TokenCredential{
//		SecretId:  secretId,
//		SecretKey: secretKey,
//		Token:     token,
//	}
//}

//func (c *TokenCredential) GetCredentialParams() map[string]string {
//	return map[string]string{
//		"SecretId": c.SecretId,
//		"Token":    c.Token,
//	}
//}
