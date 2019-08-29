package health

type ApplicationHealth interface {
	Check() error
}

func NewApplicationHealthFunc(f func() error) ApplicationHealth {
	return applicationHealthCheckerFunc(f)
}

type applicationHealthCheckerFunc func() error

func (f applicationHealthCheckerFunc) Check() error {
	return f()
}
