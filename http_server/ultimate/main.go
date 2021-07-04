package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	MaxWorker = 100 // 工作池
	MaxQueue  = 200
)

type Payload struct {
	Id int
}

type Job struct {
	Payload Payload
}

type Worker struct {
	WorkerPool chan chan Job
	JobChannel chan Job
	quit       chan bool
}

// 一个可以发送工作请求的缓冲channel
var JobQueue chan Job

func NewWorker(workerPool chan chan Job) Worker {
	return Worker{
		WorkerPool: workerPool,
		JobChannel: make(chan Job),
		quit:       make(chan bool),
	}
}

// Start 方法开启一个 worker 循环监听退出 channel
// 可按需停止这个循环
func (w *Worker) Start() {
	go func() {
		for {
			// 将当前的worker 注册到worker队列中
			w.WorkerPool <- w.JobChannel
			select {
			case job := <-w.JobChannel:
				// 真正执行业务的地方
				// 模拟操作的耗时
				time.Sleep(500 * time.Millisecond)
				fmt.Printf("上传成功:%v\n", job)
			case <-w.quit:
				return
			}
		}
	}()
}

func (w *Worker) stop() {
	go func() {
		w.quit <- true
	}()
}

// 初始化操作
type Dispatcher struct {
	// 注册到 dispatcher 的 worker channel 池
	WorkerPool chan chan Job
}

func NewDispatcher(maxWorkers int) *Dispatcher {
	pool := make(chan chan Job, maxWorkers)
	return &Dispatcher{WorkerPool: pool}
}

func (d *Dispatcher) Run(workerNum int) {
	// 开始运行N个worker
	for i := 0; i < workerNum; i++ {
		worker := NewWorker(d.WorkerPool)
		worker.Start()
	}
}

func (d *Dispatcher) dispatch() {
	for {
		select {
		case job := <-JobQueue:
			go func(job Job) {
				// 阐释获取一个可用的worker job channel, 阻塞直到有可用worker
				jobChannel := <-d.WorkerPool
				// 分发任务到 worker job channel 中
				jobChannel <- job
			}(job)
		}
	}
}

// 往工作队列中写入一个工作
func payloadHandler(w http.ResponseWriter, r *http.Request) {
	work := Job{Payload: Payload{Id: 123}}
	JobQueue <- work
	_, _ = w.Write([]byte("操作成功"))
}

func init() {
	// 初始化工作地队列
	JobQueue = make(chan Job, MaxQueue)
}

func main() {
	// 通过调度器创建worker，监听来之JobQueue的任务
	d := NewDispatcher(MaxWorker)
	d.Run(MaxWorker)
	http.HandleFunc("/payload", payloadHandler)
	log.Fatal(http.ListenAndServe(":8099", nil))
}
