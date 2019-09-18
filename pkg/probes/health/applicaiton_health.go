package health

//ApplicationHealth represents health checker for application
type ApplicationHealth interface {
	Check() error
}

//NewApplicationHealthFunc adapt specific func and returns ApplicationHealth
func NewApplicationHealthFunc(f func() error) ApplicationHealth {
	return applicationHealthCheckerFunc(f)
}

type applicationHealthCheckerFunc func() error

func (f applicationHealthCheckerFunc) Check() error {
	return f()
}
