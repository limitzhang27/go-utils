package sync

import (
	"sync"
	"time"
)

type Wait struct {
	wg sync.WaitGroup
}

func (w *Wait) Add(delta int) {
	w.wg.Add(delta)
}
func (w *Wait) Done() {
	w.wg.Done()
}

func (w *Wait) Wait() {
	w.wg.Wait()
}

// return isTimeout
func (w *Wait) WaitWithTimeout(timeout time.Duration) bool {
	c := make(chan bool)
	go func() {
		defer close(c)
		w.wg.Wait() // 等待当前wait group结束
		c <- true
	}()
	select {
	// 要么wait group结束，要么时间到
	case <-c:
		return false
	case <-time.After(timeout):
		return true
	}
}
