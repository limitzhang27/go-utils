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
	s <- struct{}{}
}

func (s Semaphore) Unlock() {
	// 为其他信号腾出空间
	<-s
}

// 实现互斥锁
type Mutex Semaphore

func NewMutex() Mutex {
	return Mutex(NewSemaphore(1))
}

// 读写锁
type RWMutex struct {
	write   chan struct{}
	readers chan int
}

func NewLock() RWMutex {
	return RWMutex{
		// 用来实现一个普通的互斥锁
		write: make(chan struct{}, 1),
		// 用来保护读锁的数量，获取读锁时通过接受通道里的值确保
		// 其他goroutine不会在同一个时间更改读锁的数量
		readers: make(chan int, 1),
	}
}

func (l RWMutex) Lock() {
	l.write <- struct{}{}
}

func (l RWMutex) Unlock() {
	<-l.write
}

func (l RWMutex) RLock() {
	// 统计当前锁的数量， 默认为0
	var rs int
	select {
	case l.write <- struct{}{}:
		// 如果write通道能发送成功，证明这是第一个加读锁的
		// 向write通道发送一个值，防止出现并发的读-写
	case rs = <-l.readers:
		// 能从通道里接到的值，证明RWMutex上已经有读锁了，下面会更新读锁
	}
	// 如果执行了 l.write <- struct{}{}, rs的值会是0
	rs++
	// 更新RWMutex读锁数量
	l.readers <- rs
}

func (l RWMutex) RUnlock() {
	// 读出读锁数量然后减一
	rs := <-l.readers
	rs--
	// 如果释放读锁后读锁的数量变为0了，抽空write通道，让write通道变为可用
	if rs == 0 {
		<-l.write
		return
	}
	// 如果释放读锁的数量减一后不是0，把新的读锁数量发送给readers通道
	l.readers <- rs
}

// waitGroup

type generation struct {
	// 用于让等待着阻塞的通道
	// 这个通道永远不会用于发送，只能用于接收和close.
	wait chan struct{}
	// 计数器， 标记需要等待执行完成的job数量
	n int
}

func newGeneration() generation {
	return generation{
		wait: make(chan struct{}),
	}
}

func (g generation) end() {
	// close通道关闭将释放因为接收通道而阻塞的goroutine
	close(g.wait)
}

// 这里我们使用一个通道来保护当前的generation
// 它基本上是WaitGroup状态的互斥量
type WaitGroup chan generation

func NewWaitGroup() WaitGroup {
	wg := make(WaitGroup, 1)
	g := newGeneration()
	// 一个新的WaitGroup上Wait，因为计数器是0，会立即返回不会阻塞线程
	// 他表现跟当前世代已经结束一样，所以这里先把世代里的Wait通道close掉
	// 防止刚创建WaitGroup时调用Wait会阻塞线程
	g.end()
	wg <- g // 一个一开始就是关闭的通道
	return wg
}

func (wg WaitGroup) Add(delta int) {
	// 当前当前的世代
	g := <-wg
	if g.n == 0 {
		// 计数器是0，创建一个新的世代
		g = newGeneration()
	}
	g.n += delta
	if g.n < 0 {
		// 跟sync库里的WaitGroup一样，不允许计数器为负数
		panic("negative WaitGroup count")
	}
	if g.n == 0 {
		// 计数器糊掉0了，关闭Wait通道，被WaitGroup的Wait方法
		// 阻塞住的线程会被释放出来继续往下执行
		g.end()
	}
	// 将更新后的世代发送回WaitGroup通道
}

func (wg WaitGroup) Done() {
	wg.Add(-1)
}

func (wg WaitGroup) Wait() {
	// 获取当前世代
	g := <-wg
	// 保存一个世代里wait通道的引用
	wait := g.wait

	// 将世代写会通道
	wg <- g
	// 接受世代里的wait通道
	// 因为wait通道里没有值，会把调用wait方法的goroutine阻塞住
	// 直到WaitGroup的计数器回到0，wait通道被close才会接触阻塞
	<-wait
}
