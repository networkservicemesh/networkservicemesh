package serialize

import "runtime"

type Executor interface {
	AsyncExec(func())
	SyncExec(func())
}

type executor struct {
	execCh      chan func()
	finalizedCh chan struct{}
}

func NewExecutor() Executor {
	rv := &executor{
		execCh:      make(chan func(), 100),
		finalizedCh: make(chan struct{}),
	}
	go rv.eventLoop()
	runtime.SetFinalizer(rv, func(f *executor) {
		close(f.finalizedCh)
	})
	return rv
}

func (t *executor) eventLoop() {
	for {
		select {
		case exec := <-t.execCh:
			exec()
		case <-t.finalizedCh:
			break
		}
	}
}

func (t *executor) AsyncExec(exec func()) {
	t.execCh <- exec
}

func (t *executor) SyncExec(exec func()) {
	done := make(chan struct{})
	t.execCh <- func() {
		exec()
		close(done)
	}
	<-done
}
