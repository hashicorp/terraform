package consumer

//go:generate hel --type TokenRefresher --output mock_token_refresher_test.go

type TokenRefresher interface {
	RefreshAuthToken() (token string, authError error)
}

func (c *Consumer) RefreshTokenFrom(tr TokenRefresher) {
	c.refresherMutex.Lock()
	defer c.refresherMutex.Unlock()

	c.refreshTokens = true
	c.tokenRefresher = tr
}

func (c *Consumer) getToken() (string, error) {
	c.refresherMutex.RLock()
	defer c.refresherMutex.RUnlock()

	return c.tokenRefresher.RefreshAuthToken()
}
