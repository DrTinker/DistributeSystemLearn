package basicpaxos

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
)

type Acceptor struct {
	// acceptor作为服务端，需要暴露端口
	listener net.Listener
	// 服务器编号
	id int
	// 承诺的提案编号
	minProposal int
	// 接受的提案编号
	acceptedNumber int
	// 接受的value
	acceptedValue interface{}
	// Learner id列表
	learnerList []int
}

func NewAcceptor(id int, learners []int) *Acceptor {
	return &Acceptor{
		id:          id,
		learnerList: learners,
	}
}

func (a *Acceptor) start() {
	// 创建server
	server := rpc.NewServer()
	// 注册server
	server.Register(a)
	// 监听端口
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.id))
	if err != nil {
		log.Fatalf("Start acceptor %d err: %v", a.id, err)
	}
	a.listener = l
	// keep alive
	go func() {
		for {
			conn, err := a.listener.Accept()
			// 重试
			if err != nil {
				continue
			}
			// 坑 不是rpc.ServeConn(conn) 而是sever.ServeConn(conn)
			go server.ServeConn(conn)
		}
	}()
}

func (a *Acceptor) close() error {
	return a.listener.Close()
}

// prepare阶段
func (a *Acceptor) Promise(args *MsgArgs, reply *MsgReply) error {
	// 判断提案的编号是否为当前见过的最大编号
	if args.Number >= a.minProposal {
		reply.OK = true
		// 是否接受过其他提案
		if a.acceptedNumber != 0 {
			// 接受过则返回
			reply.Number = a.acceptedNumber
			reply.Value = a.acceptedValue
		}
	} else {
		reply.OK = false
	}

	return nil
}

// accept阶段
func (a *Acceptor) Accept(args *MsgArgs, reply *MsgReply) error {
	// 判断是否accept中的编号是否还是最大
	if args.Number > a.minProposal {
		// 接受提案
		reply.OK = true
		a.acceptedNumber = args.Number
		a.acceptedValue = args.Value
		a.minProposal = args.Number
		// 转发给learner
		for _, lid := range a.learnerList {
			go func(lid int) {
				args.From = a.id
				args.To = lid
				resp := new(MsgReply)
				call(fmt.Sprintf("127.0.0.1:%d", lid), "Learner.Learn", args, resp)
			}(lid)
		}
	} else {
		// 拒绝提案
		reply.OK = false
	}

	return nil
}
