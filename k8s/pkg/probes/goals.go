package probes

import "sync/atomic"

//Goals provides simple API for manage goals
type Goals interface {
	//Done means done one of goals
	Done()
	//IsComplete checks all goals have done
	IsComplete() bool
	//TODO returns the number of remaining goals
	TODO() int
}

//NewGoals creates a new instance of Goals with specified count
func NewGoals(goalCount int) Goals {
	return &goals{
		count: int32(goalCount),
	}
}

type goals struct {
	count int32
}

func (g *goals) Done() {
	val := atomic.AddInt32(&g.count, -1)
	if val < 0 {
		panic("incorrect goal count")
	}
}

func (g *goals) IsComplete() bool {
	return atomic.LoadInt32(&g.count) == 0
}

func (g *goals) TODO() int {
	return int(atomic.LoadInt32(&g.count))
}
