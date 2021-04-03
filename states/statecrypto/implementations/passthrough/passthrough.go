package passthrough

type PassthroughStateWrapper struct {
}

func (p *PassthroughStateWrapper) Encrypt(data []byte) ([]byte, error) {
	return data, nil
}

func (p *PassthroughStateWrapper) Decrypt(data []byte) ([]byte, error) {
	return data, nil
}
