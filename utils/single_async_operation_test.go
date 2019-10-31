package utils

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/onsi/gomega"
)

func TestUsingSingleAsyncOperation(t *testing.T) {
	defer func() {
		err := recover()
		if err != nil {
			logrus.Errorf("test should not fails, err: %v", err)
			t.FailNow()
		}
	}()
	m := map[int]int{}
	op := NewSingleAsyncOperation(func() {
		m[0] = 0
		m[1] = 1
		m[2] = 2
	})
	for i := 0; i < 10000; i++ {
		op.Run() // check on concurrent modification
	}
}

func TestSingleOperationRun(t *testing.T) {
	ch := make(chan bool)
	op := NewSingleAsyncOperation(func() {
		ch <- true
	})
	op.Run()
	select {
	case <-ch:
		return
	case <-time.After(time.Second):
		t.Fatal("op did not run")
	}
}

func TestLongSingleOperationRun(t *testing.T) {
	assert := gomega.NewWithT(t)
	longOperation := make(chan struct{})
	counter := int32(0)
	testDuration := time.Second
	op := NewSingleAsyncOperation(func() {
		atomic.AddInt32(&counter, 1)
		<-longOperation
	})
	timeElapsed := false
	timeCh := time.After(testDuration / 2)
	for !timeElapsed {
		select {
		case <-timeCh:
			timeElapsed = true
		default:
			op.Run()
			<-time.After(time.Millisecond * 25)
		}
	}
	close(longOperation)
	op.Wait()
	assert.Expect(counter).Should(gomega.Equal(int32(2)))
}

func TestLongSingleOperationRerun(t *testing.T) {
	assert := gomega.NewWithT(t)
	block := make(chan struct{})
	counter := int32(0)
	op := singleAsyncOperation{
		body: func() {
			if atomic.AddInt32(&counter, 1) > 2 {
				t.FailNow()
			}
			<-block
		},
		state: notScheduled,
	}

	op.Run()
	assert.Expect(op.state).Should(gomega.Equal(running))
	op.Run()
	assert.Expect(op.state).Should(gomega.Equal(scheduledAndRunning))
	op.Run()
	assert.Expect(op.state).Should(gomega.Equal(scheduledAndRunning))
	close(block)
	op.Wait()
	assert.Expect(op.state).Should(gomega.Equal(notScheduled))
	assert.Expect(counter).Should(gomega.Equal(int32(2)))
}
