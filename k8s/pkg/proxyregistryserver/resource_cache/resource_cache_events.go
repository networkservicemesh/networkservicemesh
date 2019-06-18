package resource_cache

type resourceEvent interface {
	accept(config cacheConfig)
}
type resourceAddEvent struct {
	source interface{}
}

func (e resourceAddEvent) accept(config cacheConfig) {
	if config.resourceAddedFunc != nil {
		config.resourceAddedFunc(e.source)
	}
}

type resourceDeleteEvent struct {
	key string
}

func (e resourceDeleteEvent) accept(config cacheConfig) {
	if config.resourceAddedFunc != nil {
		config.resourceDeletedFunc(e.key)
	}
}

type resourceUpdateEvent struct {
	source interface{}
}

func (e resourceUpdateEvent) accept(config cacheConfig) {
	if config.resourceUpdatedFunc != nil {
		config.resourceUpdatedFunc(e.source)
	}
}

type syncExecEvent struct {
	toExec func()
	doneCh chan<- struct{}
}

func (e syncExecEvent) accept(config cacheConfig) {
	e.toExec()
	close(e.doneCh)
}

type resourceGetEvent struct {
	key   string
	getCh chan<- interface{}
}

func (e resourceGetEvent) accept(config cacheConfig) {
	if config.resourceGetFunc != nil {
		e.getCh <- config.resourceGetFunc(e.key)
	}
}
