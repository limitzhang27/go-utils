package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

/*
这是一个初级的http处理器
在高并发的场景下，不对goroutine数进行控制
CPU使用率爆张，内存暂用爆张，直到程序崩溃

如果此操作落地至数据库，例如mysql，那么相应的，
数据库的服务器I/O、网络带宽、CPU负载、内存消耗都会达到非常高的情况，一并崩溃
所以，一旦程序中出现不可控制的事务，往往是危险的信号
*/

type Payload struct {
	Id int
}

func (p *Payload) UpdateToS3() error {
	// 存储逻辑，模拟操作耗时
	time.Sleep(500 * time.Millisecond)
	fmt.Println("上传成功")
	return nil
}

func payloadHandler(w http.ResponseWriter, r *http.Request) {
	// 业务过滤
	// 请求body解析
	p := Payload{Id: 1}
	go func() {
		_ = p.UpdateToS3()
	}()
	_, _ = w.Write([]byte("操作成功"))
}

func main() {
	http.HandleFunc("/payload", payloadHandler)
	log.Fatal(http.ListenAndServe(":8099", nil))
}
