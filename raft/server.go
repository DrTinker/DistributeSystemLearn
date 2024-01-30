package raft

import (
	"fmt"
	"log"
	"net"
	"net/rpc"

	"github.com/sirupsen/logrus"
)

type Server struct {
	id       int
	listener net.Listener
}

func NewServer() {

}

// 开启服务
func (s *Server) Serve() {
	// 创建server
	server := rpc.NewServer()
	// 注册server
	server.RegisterName("Raft", s)
	// 监听端口
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", s.id))
	if err != nil {
		logrus.Errorf("Start raft node %d err: %v", s.id, err)
	}
	s.listener = l
	// keep alive
	go func() {
		for {
			conn, err := s.listener.Accept()
			// 重试
			if err != nil {
				continue
			}
			// 坑 不是rpc.ServeConn(conn) 而是sever.ServeConn(conn)
			go server.ServeConn(conn)
		}
	}()
}

func call(id int, method string, args, reply interface{}) bool {
	// TODO 暂时这样写，后面替换
	addr := fmt.Sprintf("127.0.0.1:%d", id)
	// 建立tcp链接
	c, err := rpc.Dial("tcp", addr)
	if err != nil {
		log.Fatalf("Call %s %s err: %v\n", addr, method, err)
		return false
	}
	defer c.Close()

	// 调用方法
	err = c.Call(method, args, reply)
	if err != nil {
		log.Fatalf("Call %s %s err: %v\n", addr, method, err)
		return false
	}

	return true
}
