package basicpaxos

import (
	"fmt"
)

type Proposer struct {
	// 服务器id
	id int
	// 轮次
	round int
	// 提案编号
	number int
	// acceptor id列表
	acceptorList []int
}

func NewProposer(id int, acceptors []int) *Proposer {
	return &Proposer{
		id:           id,
		acceptorList: acceptors,
	}
}

func (p *Proposer) Propose(value interface{}) interface{} {
	// 开启新的轮次
	p.round++
	p.number = p.proposalNumber()
	// 阶段1
	// 向全部acceptors广播提案，但是不发送提案value
	promiseCnt := 0 // 计算收到多少promise
	maxNumber := 0  // 如果acceptor之前接收过别的提案，需要在promise中返回接受的value，这个参数目的在于选出最新版的value
	for _, aid := range p.acceptorList {
		// 封装msg
		args := &MsgArgs{
			Number: p.number,
			From:   p.id,
			To:     aid,
		}
		reply := &MsgReply{}
		// rpc调用
		ok := call(fmt.Sprintf("127.0.0.1:%d", aid), "Acceptor.Promise", args, reply)
		// 调用失败忽视
		if !ok {
			continue
		}
		// 本次调用收到promise
		if reply.OK {
			// 如果reply.Number不是0，说明接受过其他提案
			promiseCnt++
			// 这里要选出acceptors接收的最新的提案value
			if reply.Number > maxNumber {
				maxNumber = reply.Number
				// 决定value
				value = reply.Value
			}
		}
		// 如果达到多数，则结束阶段1
		if promiseCnt == p.majority() {
			break
		}
	}

	// 阶段2
	// 如果收到超过半数的promise，则向所有acceptors发送决定好的value
	if promiseCnt < p.majority() {
		return nil
	}
	// 向acceptors广播args，这次带value
	acceptCnt := 0
	for _, aid := range p.acceptorList {
		// 封装msg
		args := &MsgArgs{
			Number: p.number,
			From:   p.id,
			To:     aid,
			Value:  value,
		}
		reply := &MsgReply{}
		// rpc调用
		ok := call(fmt.Sprintf("127.0.0.1:%d", aid), "Acceptor.Accept", args, reply)
		// 调用失败忽视
		if !ok {
			continue
		}
		// 计算收到的accept
		if reply.OK {
			acceptCnt++
		}
		// 超过半数说明本次提案成功
		if acceptCnt >= p.majority() {
			return value
		}
	}

	return nil
}

func (p *Proposer) majority() int {
	return len(p.acceptorList)/2 + 1
}

func (p *Proposer) proposalNumber() int {
	return p.round<<16 | p.id
}
