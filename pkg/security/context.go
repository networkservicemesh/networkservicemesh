package security

type Context interface {
	GetResponseOboToken() string
	SetResponseOboToken(token string)
}

type contextImpl struct {
	responseOboToken string
}

func NewContext() Context {
	return &contextImpl{}
}

func (c *contextImpl) GetResponseOboToken() string {
	return c.responseOboToken
}

func (c *contextImpl) SetResponseOboToken(token string) {
	c.responseOboToken = token
}
