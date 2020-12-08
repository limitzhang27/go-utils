package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
)

func main() {
	server, err := net.Listen("tcp", ":1080")
	if err != nil {
		log.Printf("Listen failed: (%v)\f", err)
		return
	}
	for {
		client, err := server.Accept()
		if err != nil {
			log.Printf("Accept failed: (%v) \n", err)
			continue
		}
		go process(client)
	}
}

func process(client net.Conn) {
	if err := Socks5Auth(client); err != nil {
		log.Println("auth error: ", err)
		_ = client.Close()
		return
	}

	target, err := Socks5Connect(client)
	if err != nil {
		log.Println("connect error: ", err)
		_ = client.Close()
		return
	}
	Socks5ForWard(client, target)
}

func Socks5Auth(client net.Conn) (err error) {
	buf := make([]byte, 256)
	// 读取 VER 和NMETHODS
	n, err := io.ReadFull(client, buf[:2])
	if n != 2 {
		var errMsg string
		if err != nil {
			errMsg = err.Error()
		} else {
			errMsg = "buf data no equal 2"
		}
		return errors.New("reading header : " + errMsg)
	}

	ver, nMethods := int(buf[0]), int(buf[1])
	if ver != 5 {
		return errors.New("invalid version")
	}

	// 读取 METHODS 列表
	n, err = io.ReadFull(client, buf[:nMethods])
	if n != nMethods {
		var errMsg string
		if err != nil {
			errMsg = err.Error()
		} else {
			errMsg = "methods num err"
		}
		return errors.New("reading methods: " + errMsg)
	}

	n, err = client.Write([]byte{0x05, 0x00})
	if n != 2 {
		var errMsg string
		if err != nil {
			errMsg = err.Error()
		} else {
			errMsg = "Write data error"
		}
		return errors.New("write rsp err: " + errMsg)
	}
	return nil
}

func Socks5Connect(client net.Conn) (net.Conn, error) {
	buf := make([]byte, 256)
	n, err := io.ReadFull(client, buf[:4])
	if n != 4 {
		var errMsg string
		if err != nil {
			errMsg = err.Error()
		} else {
			errMsg = "header no equal 4"
		}
		return nil, errors.New("read header: " + errMsg)
	}

	ver, cmd, _, atype := buf[0], buf[1], buf[2], buf[3]
	if ver != 5 || cmd != 1 {
		return nil, errors.New("invalid ver/cmd")
	}

	addr := ""
	switch atype {
	case 1:
		n, err = io.ReadFull(client, buf[:4])
		if n != 4 {
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			} else {
				errMsg = "buf data no equal 4"
			}
			return nil, errors.New("invalid IPv4: " + errMsg)
		}

		addr = fmt.Sprintf("%d.%d.%d.%d", buf[0], buf[1], buf[2], buf[3])
	case 3:
		n, err = io.ReadFull(client, buf[:1])
		if n != 4 {
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			} else {
				errMsg = "hostnameLen data no equal 1"
			}
			return nil, errors.New("invalid hostname: " + errMsg)
		}
		addrLen := int(buf[0])

		n, err = io.ReadFull(client, buf[:addrLen])
		if n != addrLen {
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			} else {
				errMsg = fmt.Sprintf("hostnameLen no equal %d", addrLen)
			}
			return nil, errors.New("invalid hostname: " + errMsg)
		}
		addr = string(buf[:addrLen])
	case 4:
		return nil, errors.New("IPv6: no supported yet")
	default:
		return nil, errors.New("invalid atyp")
	}

	n, err = io.ReadFull(client, buf[:2])
	if n != 2 {
		var errMsg string
		if err != nil {
			errMsg = err.Error()
		} else {
			errMsg = "port len no equal 2"
		}
		return nil, errors.New("read port: " + errMsg)
	}
	port := binary.BigEndian.Uint16(buf[:2])

	// addr 和 port 都就位了，马上创建一个dst的链接
	destAddrPort := fmt.Sprintf("%s:%d", addr, port)
	dest, err := net.Dial("tcp", destAddrPort)
	if err != nil {
		return nil, errors.New("dial dst: " + err.Error())
	}
	n, err = client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	if err != nil {
		_ = dest.Close()
		return nil, errors.New("write rsp: " + err.Error())
	}
	return dest, nil
}

func Socks5ForWard(client, target net.Conn) {
	forward := func(src, dest net.Conn) {
		defer func() {
			_ = src.Close()
		}()

		defer func() {
			_ = dest.Close()
		}()

		_, _ = io.Copy(src, dest)
	}
	go forward(client, target)
	go forward(target, client)
}
