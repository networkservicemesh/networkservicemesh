package health

import "sync"

//Appender appends health checkers
type Appender interface {
	Append(health ApplicationHealth)
	Iterate(func(health ApplicationHealth) bool)
}

//AppenderImpl implementation of interface health.Appender.
type AppenderImpl struct {
	heaths []ApplicationHealth
	mutex  sync.Mutex
}

//Iterate accepts function to iterable
func (c *AppenderImpl) Iterate(f func(health ApplicationHealth) bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, checker := range c.heaths {
		if !f(checker) {
			return
		}
	}
}

//Append adds health checker
func (c *AppenderImpl) Append(health ApplicationHealth) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.heaths = append(c.heaths, health)
}
