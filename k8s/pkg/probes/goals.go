package probes

//Goals provides simple API for manage goalsImpl
type Goals interface {
	IsComplete() bool
	TODO() string
}
