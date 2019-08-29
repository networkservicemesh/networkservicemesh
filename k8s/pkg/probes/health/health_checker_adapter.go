package health

import "sync"

type Appender interface {
	Append(health ApplicationHealth)
	Iterate(func(health ApplicationHealth) bool)
}

type AppenderImpl struct {
	heaths []ApplicationHealth
	mutex  sync.Mutex
}

func (c *AppenderImpl) Iterate(f func(health ApplicationHealth) bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, checker := range c.heaths {
		if !f(checker) {
			return
		}
	}
}

func (c *AppenderImpl) Append(health ApplicationHealth) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.heaths = append(c.heaths, health)
}
