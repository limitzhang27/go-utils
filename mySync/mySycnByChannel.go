package mySync

// channel 实现sync.Once
type Once chan struct{}

func NewOnce() Once {
	o := make(Once, 1)
	// 只允许一个goroutine接收，其他goroutine会被阻塞
	o <- struct{}{}
	return o
}

func (o Once) Do(f func()) {
	_, ok := <-o
	if !ok {
		// channel已经被关闭了
		// 证明f已经被执行过了，直接return
		return
	}

	// 调用f, 因为channel中只有一个值
	// 所以只有一个goroutine会到达这里
	f()
	// 关闭通道，这里江辉使用所以所有的等待
	// 以及未来会调用Do方法的goroutine
	close(o)
}


// channel 实现信号量
type Semaphore chan struct{}

func NewSemaphore(size int) Semaphore {
	return make(Semaphore, size)
}

func (s Semaphore) Lock() {
	// 只有在s还有空间的时候才能发送成功
	s <- struct {}{}
}

func (s Semaphore) Unlock() {
	// 为其他信号腾出空间
	<- s
}


// 实现互斥锁
type Mutex Semaphore

func NewMutex() Mutex {
	return Mutex(NewSemaphore(1))
}

