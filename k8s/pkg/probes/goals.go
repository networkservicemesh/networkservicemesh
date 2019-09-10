package probes

//Goals provides simple API for manage goals
type Goals interface {
	IsComplete() bool
	Status() string
}
