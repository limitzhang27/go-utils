package pool

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

var (
	ErrMaxActiveConnReached = errors.New("MaxActiveConnReached")
	ErrConnectionIsNil = errors.New("connection is nil. rejecting")
)

// Config 连接池相关配置
type Config struct {
	// 连接池中拥有的最小连接数
	InitialCap int
	// 最大并发存活连接数
	MaxCap int
	// 最大空闲连接
	MaxIdle int
	// 生成连接的方法
	Factory func() (interface{}, error)
	// 关闭链接的方法
	Close func(interface{}) error
	// 检查连接是否有效的方法
	Ping func(interface{}) error
	// 连接最大空闲时间， 超过改时间则将失效
	IdleTimeout time.Duration
}

// 封装连接实例
type idleConn struct {
	conn interface{}
	t time.Time
}

// 等到队列中的实例
type connReq struct {
	idleConn *idleConn
}

// channelPool
type channelPool struct {
	// 互斥锁
	mu 				sync.RWMutex
	// 连接实例
	conns 			chan *idleConn
	// 创建连接方法
	factory 		func() (interface{}, error)
	// 关闭连接方法
	close 			func(interface{}) error
	// 测试连接方法
	ping 			func(interface{}) error
	// 连接失效时长
	idleTimeout 	time.Duration
	// 等待超时时长
	waitTimeout 	time.Duration
	// 最大连接数量
	maxActive 		int
	// 已打开连接的数量
	openingConns	int
	// 等待拿到实例队列
	connReqs 		[]chan connReq
}

// 初始化连接池
func NewChannelPool(poolConfig *Config) (Pool, error) {
	// 配置参数错误
	if !(poolConfig.InitialCap <= poolConfig.MaxIdle && poolConfig.MaxCap >= poolConfig.MaxIdle &&
		poolConfig.InitialCap >= 0) {
		return nil, errors.New("invalid capacity settings")
	}

	if poolConfig.Factory == nil {
		return nil, errors.New("invalid factory func setting")
	}

	if poolConfig.Close == nil {
		return nil, errors.New("invalid close func settings")
	}

	c := &channelPool{
		conns: 			make(chan *idleConn, poolConfig.MaxIdle),
		factory: 		poolConfig.Factory,
		close:			poolConfig.Close,
		idleTimeout: 	poolConfig.IdleTimeout,
		maxActive: 		poolConfig.MaxIdle,
		openingConns: 	poolConfig.InitialCap,
	}

	if  poolConfig.Ping != nil {
		c.ping	= poolConfig.Ping
	}

	for i := 0; i < poolConfig.InitialCap; i++ {
		conn, err := c.factory()
		if err != nil {
			c.Release()
			return nil, fmt.Errorf("factory is not able to fill the pool: %s", err)
		}
		c.conns <- &idleConn{conn: conn, t: time.Now()}
	}
	return c, nil
}

// getConns 获取所有连接
func (c *channelPool) getConns() chan *idleConn  {
	c.mu.Lock()
	conns := c.conns
	c.mu.Unlock()
	return conns
}

// Get 从pool中取一个连接
func (c *channelPool) Get() (interface{}, error) {
	conns := c.getConns()
	if conns == nil {
		return nil, ErrClosed
	}
	for {
		select {
		case wrapConn := <-conns:
			if wrapConn == nil {
				return nil, ErrClosed
			}
			// 判断是否超时， 超时则丢弃
			// (重新赋值是为了防止这个变量突然变成0，导致下面的时间判断出错)
			if  timeout := c.idleTimeout; timeout > 0 {
				if wrapConn.t.Add(timeout).Before(time.Now()) {
					// 丢弃并关闭该连接
					_ = c.Close(wrapConn)
					continue
				}
			}
			// 判断是否失效，失效则丢弃，如果用户没有设定ping方法，就不检查
			if err := c.Ping(wrapConn.conn); err != nil {
				continue
			}
			return wrapConn.conn, nil
		default:
			c.mu.Lock()
			log.Printf("openConn %v %v", c.openingConns, c.maxActive)
			// 当正在连接的数量大于最大连接数, 加入等待队列中
			if c.openingConns >= c.maxActive {
				req := make(chan connReq, 1)
				c.connReqs = append(c.connReqs, req)
				c.mu.Unlock()
				ret, ok := <- req
				if !ok { // 等待队列异常
					return nil, ErrMaxActiveConnReached
				}
				if timeout := c.idleTimeout; timeout > 0 {
					if ret.idleConn.t.Add(timeout).Before(time.Now()) {
						// 等到的连接已经超时了，关闭它
						_ = c.close(ret.idleConn.conn)
						continue
					}
				}
				return ret.idleConn.conn, nil
			}
			if c.factory == nil {
				c.mu.Unlock()
				return nil, ErrClosed
			}
			conn, err := c.factory()
			if err != nil {
				return nil, err
			}
			c.openingConns++
			c.mu.Unlock()
			return conn, nil
		}
	}
}

func (c *channelPool) Put(conn interface{}) error {
	if conn == nil {
		 return ErrConnectionIsNil
	}

	c.mu.Lock()

	if c.conns == nil {
		c.mu.Unlock()
		return c.Close(conn)
	}

	// 等待队列不为空
	if l := len(c.connReqs); l > 0 {
		req := c.connReqs[0]
		copy(c.connReqs, c.connReqs[1:])
		c.connReqs = c.connReqs[:l-1]
		req <- connReq{
			idleConn: &idleConn{conn: conn, t:    time.Now()},
		}
		c.mu.Unlock()
		return nil
	} else {
		select {
		// 保存在空闲连接池
		case c.conns <- &idleConn{conn: conn, t: time.Now()}:
			c.mu.Unlock()
			return nil
		default:	// 空闲连接池满了的话，直接关闭
			c.mu.Unlock()
			return c.close(conn)

		}
	}
}


// Close 关闭链接
func (c *channelPool) Close(conn interface{}) error {
	if conn == nil {
		return ErrConnectionIsNil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.close == nil {
		return nil
	}
	c.openingConns--
	return c.close(conn)
}

// Ping 检查单挑链接是否有效
func (c *channelPool) Ping(conn interface{}) error {
	if conn == nil {
		return ErrConnectionIsNil
	}
	if c.ping == nil {
		return nil
	}
	return c.ping(conn)
}

func (c *channelPool) Release()  {
	c.mu.Lock()
	conns := c.conns
	c.conns = nil
	c.factory = nil
	c.ping = nil
	closeFun := c.close
	c.close = nil
	c.mu.Unlock()

	if conns == nil {
		return
	}

	close(conns)
	for wrapconn := range conns {
		_ = closeFun(wrapconn.conn)
	}
}

func (c *channelPool) Len() int {
	return len(c.getConns())
}