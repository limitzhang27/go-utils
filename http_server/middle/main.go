package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

/*
这一版借助buffered的channel来完成这个功能，
这样控制住了无限制的goroutine，但是依然没有解决问题

处理请求是一个同步的操作，每次只会处理一个任务，
然而高并发下请求进来的速度远远超过了处理的速度，这种情况，
一旦channel满了之后，后续的请求将会阻塞的等待，
相应的时间会大幅度的开始增加，甚至不再有任何的相应。
*/

const MaxQueue = 400

var Queue chan Payload

type Payload struct {
	Id int
}

func init() {
	Queue = make(chan Payload, MaxQueue)
}

func (p *Payload) UpdateToS3() error {
	// 存储逻辑，模拟操作耗时
	time.Sleep(500 * time.Millisecond)
	fmt.Println("上传成功")
	return nil
}

func payloadHandler(w http.ResponseWriter, r *http.Request) {
	// 业务过滤
	// 请求体body解析
	p := Payload{
		Id: 1,
	}
	Queue <- p
	_, _ = w.Write([]byte("操作成功"))
}

// 处理任务
func StartProcessor() {
	for {
		select {
		case payload := <-Queue:
			_ = payload.UpdateToS3()
		}
	}
}

func main() {
	http.HandleFunc("/payload", payloadHandler)
	// 单独开一个g接收与处理任务
	go StartProcessor()
	log.Fatal(http.ListenAndServe(":8099", nil))
}
