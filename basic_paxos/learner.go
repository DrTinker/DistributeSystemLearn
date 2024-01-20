package basicpaxos

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
)

type Learner struct {
	// 作为服务端的监听端口
	listener net.Listener
	// 服务器id
	id int
	// 接受提案map
	acceptMsgMap map[int]*MsgArgs
}

func NewLearner(id int, acceptors []int) *Learner {
	acceptorMap := make(map[int]*MsgArgs, len(acceptors))
	for _, aid := range acceptors {
		acceptorMap[aid] = &MsgArgs{
			Number: 0,
			Value:  nil,
		}
	}
	return &Learner{
		id:           id,
		acceptMsgMap: acceptorMap,
	}
}

func (l *Learner) start() {
	// 创建server
	server := rpc.NewServer()
	// 注册server
	server.Register(l)
	// 监听端口
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", l.id))
	if err != nil {
		log.Fatalf("Start Learner %d err: %v", l.id, err)
	}
	l.listener = lis
	// keep alive
	go func() {
		for {
			conn, err := l.listener.Accept()
			// 重试
			if err != nil {
				continue
			}
			go server.ServeConn(conn)
		}
	}()
}

func (l *Learner) close() error {
	return l.listener.Close()
}

func (l *Learner) Learn(args *MsgArgs, reply *MsgReply) error {
	// 根据提案编号判断是否学习
	acc := l.acceptMsgMap[args.From]
	// acceptor发来的是最新提案
	if acc.Number < args.Number {
		// 保存提案
		l.acceptMsgMap[args.From] = args
		reply.OK = true
	} else {
		reply.OK = false
	}
	return nil
}

// 接收超过半数的提案
func (l *Learner) chosen() interface{} {
	acceptCntMap := map[int]int{}
	// learner中的map是以服务器id为key，这里需要提案number为key
	acceptMap := map[int]*MsgArgs{}

	for _, acc := range l.acceptMsgMap {
		// 如果是被接受的提案
		if acc.Number != 0 {
			acceptCntMap[acc.Number]++
			acceptMap[acc.Number] = acc
		}
	}

	// 选出超过半数的提案
	for number, cnt := range acceptCntMap {
		if cnt >= l.majority() {
			return acceptMap[number].Value
		}
	}

	return nil
}

func (l *Learner) majority() int {
	return len(l.acceptMsgMap)/2 + 1
}
