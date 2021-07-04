package server

import (
	"godis/lib/sync/atomic"
	"sync"
)

var (
	UnknownErrReplyBytes = []byte("-ERR unknown\r\n")
)

type Handler struct {
	// 正在链接中的客户端tcp
	activeConn sync.Map
	//db db.DB

	// 关闭状态
	closing atomic.AtomicBool
}
