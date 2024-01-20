package basicpaxos

import (
	"log"
	"net/rpc"
)

// rpc调用必须是导出类型
type MsgArgs struct {
	// 编号
	Number int
	// 值
	Value interface{}
	// 发送者
	From int
	// 接受者
	To int
}

type MsgReply struct {
	OK     bool
	Number int
	Value  interface{}
}

// 封装rpc调用
func call(addr, method string, args, reply interface{}) bool {
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
