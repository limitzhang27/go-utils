package wait

import (
	"sync"
	"time"
)

/*
 带有超时设置的 WaitGroup
*/

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

func (w *Wait) WaitWithTimeout(t time.Duration) bool {
	c := make(chan struct{})

	go func() {
		defer close(c)
		w.wg.Wait()
		c <- struct{}{}
	}()

	select {
	case <-c:
		return false // completed normally
	case <-time.After(t):
		return true // timeout
	}
}
