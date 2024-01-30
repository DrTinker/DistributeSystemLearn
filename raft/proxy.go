package raft

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// RPC代理，用于初始化rpc服务，模拟延迟等
type RPCProxy struct {
	raft *Raft
}

func NewRPCProxy(r *Raft) *RPCProxy {
	return &RPCProxy{
		raft: r,
	}
}

func (rpp *RPCProxy) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) error {
	if len(os.Getenv("RAFT_UNRELIABLE_RPC")) > 0 {
		dice := rand.Intn(10)
		if dice == 9 {
			logrus.Warn("drop RequestVote")
			return fmt.Errorf("RPC failed")
		} else if dice == 8 {
			logrus.Warn("delay RequestVote")
			time.Sleep(75 * time.Millisecond)
		}
	} else {
		time.Sleep(time.Duration(1+rand.Intn(5)) * time.Millisecond)
	}
	return rpp.raft.RequestVote(args, reply)
}

func (rpp *RPCProxy) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) error {
	if len(os.Getenv("RAFT_UNRELIABLE_RPC")) > 0 {
		dice := rand.Intn(10)
		if dice == 9 {
			logrus.Warn("drop AppendEntries")
			return fmt.Errorf("RPC failed")
		} else if dice == 8 {
			logrus.Warn("delay AppendEntries")
			time.Sleep(75 * time.Millisecond)
		}
	} else {
		time.Sleep(time.Duration(1+rand.Intn(5)) * time.Millisecond)
	}
	return rpp.raft.AppendEntries(args, reply)
}
