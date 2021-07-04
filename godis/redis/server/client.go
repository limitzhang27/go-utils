package server

import (
	"godis/lib/sync/atomic"
	"godis/lib/sync/wait"
	"net"
	"sync"
	"time"
)

type Client struct {
	/* 与客户端的 TCP 连接 */
	conn net.Conn

	// 带有超时的 WaitGroup, 当相应被完整发送前保持
	// 当响应被完整发送前保持 waiting 状态， 阻止链接被关闭
	waitingReply wait.Wait

	/* 标记客户端正在发送指令 */
	sending atomic.AtomicBool

	/* 客户端正在发送的参数数量，即 Array 第一行指定的数组长度 */
	expectedArgsCount uint32

	/* 已经接收的参数数量， 即 len(args) */
	receivedCount uint32

	/* 已经接收到的命令参数， 每个参数由一个[]byte 表示*/
	args [][]byte

	/* 发送时加锁 */
	mu sync.Mutex

	/* 订阅的频道 */
	subs map[string]struct{}
}

func (c *Client) Close() error {
	c.waitingReply.WaitWithTimeout(10 * time.Second)
	_ = c.conn.Close()
	return nil
}

func MakeClient(conn net.Conn) *Client {
	return &Client{conn: conn}
}

// 发送
func (c *Client) Write(b []byte) error {
	if b == nil || len(b) == 0 {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	_, err := c.conn.Write(b)
	return err
}

// 订阅操作
func (c *Client) SubsChannel(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.subs == nil {
		c.subs = make(map[string]struct{})
	}
	c.subs[channel] = struct{}{}
}

// 取消订阅
func (c *Client) UnSubsChannel(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.subs == nil {
		return
	}
	delete(c.subs, channel)
}

// 订阅数
func (c *Client) SubsCount() int {
	if c.subs == nil {
		return 0
	}
	return len(c.subs)
}

// 获取订阅的频道
func (c *Client) GetSubsChannels() []string {
	if c.subs == nil {
		return []string{}
	}
	channels := make([]string, len(c.subs))
	i := 0
	for c := range c.subs {
		channels[i] = c
		i++
	}
	return channels
}
