package main

import (
	"bufio"
	"context"
	"fmt"
	"godis/lib/sync/atomic"
	"godis/lib/sync/wait"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Config struct {
	Address string
}

type Handler interface {
	Handle(ctx context.Context, conn net.Conn)
	Close() error
}

// TCP监听服务
func ListenAndServer(cfg *Config, handler Handler) {
	// 监听TCP服务
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		log.Fatal(fmt.Sprintf("listen err: %v", err))
	}

	// 监听中断信号
	var closing atomic.AtomicBool
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		switch <-sigChan {
		case syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			// 接收中断信号
			log.Println("shutting donn...")
			// 设置标志位为关闭中，使用原子操作保证线程可见性
			closing.Set(true)
			// 先关闭 listener 防止新的连接计入
			// listener 关闭之后， listener.Accept() 会立即返回错误
			_ = listener.Close()
			// 逐个关闭已建立链接捏
			_ = handler.Close()
		}
	}()

	defer func() {
		// 在出现未知错误或panic后保证正常管理
		// 这里存在一个问题是： 在应用正常关闭后悔再次出现关闭操作
		_ = listener.Close()
		_ = handler.Close()
	}()
	log.Println(fmt.Sprintf("bind: %s, start listenning...", cfg.Address))

	ctx, _ := context.WithCancel(context.Background())

	// waitGroup的计数是当前存在的连接数
	// 进入关闭流程事，主程序应该等到所有连接关闭了再退出
	var waitDone sync.WaitGroup
	for {
		conn, err := listener.Accept()
		if err != nil {
			if closing.Get() {
				// 收到关闭信号后进入此流程，此时listener 已被监听系统信号的goroutine关闭(在上面)
				log.Println("waiting disconnect...")
				// 主协程应等待应用层服务器完成工作再关闭链接
				// 如果等待，主协程会因为 listener 的关闭而直接关闭了，
				// 当 listener 关闭之后，会去遍历所有连接并关闭它们
				waitDone.Wait()
				return
			}
			log.Println(fmt.Sprintf("accept err: %v", err))
			continue
		}
		log.Println("accept link")
		go func() {
			defer func() {
				waitDone.Done()
			}()
			// 使用waitGroup来标识当前还有连接正在执行
			waitDone.Add(1)
			handler.Handle(ctx, conn)
		}()
	}
}

// 客户端连接抽象
type Client struct {
	// tcp 连接
	Conn net.Conn

	// 带有 timeout 功能的WaitGroup， 用于优雅关闭
	// 当响应被完成发送前保持 waiting 状态，阻止连接被关闭
	Waiting wait.Wait
}

type EchoHandle struct {
	// 保存所有工作状态的client集合(把map当成set用)
	// 需要用并发安全的容器
	activeConn sync.Map

	// 和 tcp server 中作用相同的关闭状态标志位
	closing atomic.AtomicBool
}

func NewEchoHandle() *EchoHandle {
	return &EchoHandle{}
}

// 关闭客户端
func (c *Client) Close() error {
	// 等待数据发送完成或者超时
	c.Waiting.WaitWithTimeout(10 * time.Second)
	_ = c.Conn.Close()
	return nil
}

// 处理函数
func (h *EchoHandle) Handle(ctx context.Context, conn net.Conn) {
	// 如果刚到当前 listener 广告关闭了，当前连接直接关闭
	if h.closing.Get() {
		_ = conn.Close()
	}

	client := &Client{
		Conn: conn,
	}
	h.activeConn.Store(client, 1)
	reader := bufio.NewReader(client.Conn) // 将连接放入缓冲池中
	for {
		msg, err := reader.ReadString('\n') // 从缓冲池中读取数据
		if err != nil {
			if err == io.EOF { // 连接关闭
				log.Println("connection close")
				h.activeConn.Delete(client)
			} else {
				log.Println(err)
			}
			return
		}
		// 发送数据前先设置waiting状态
		client.Waiting.Add(1)

		// 模拟关闭时未完成的发送
		//time.Sleep(10 * time.Second)

		_, _ = client.Conn.Write([]byte(msg))
		client.Waiting.Done() // 发送结束
	}
}

// 遍历所有连接并执行关闭，如果当前连接正在发送消息，会等待发送结束或者超时
func (h *EchoHandle) Close() error {
	log.Println("handler shutting down...")
	h.closing.Set(true)
	h.activeConn.Range(func(key, value interface{}) bool {
		client := key.(*Client)
		_ = client.Close()
		return true
	})
	return nil
}

func main() {
	echoHandle := NewEchoHandle()
	config := &Config{Address: ":8000"}
	ListenAndServer(config, echoHandle)
}
