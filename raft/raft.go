package raft

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	LeaderType    int = 0
	CandidateType int = 1
	FollowerType  int = 2
)

// 一个Raft节点的数据结构
type Raft struct {
	// 互斥锁
	mu sync.Mutex
	// 网络模块
	// 本节点的id
	id int
	// 其他节点的id
	peers []int

	// 算法模块
	// 标识本节点当前身份
	state int
	// term + index 唯一定位一条日志
	currentTerm int
	log         []*LogEntry
	// 记录投票信息
	voteFor int
	// 选举/心跳超时计时器
	electionTime time.Time
}

// 一个Log的结构，通过 term + index 唯一标识一条日志
type LogEntry struct {
	term  int
	index int
	value interface{}
}

func NewRaftNode() *Raft {
	return &Raft{}
}

// 是否超过法定人数
func (r *Raft) isQuorum(n int) bool {
	return n >= ((len(r.peers) + 1) / 2)
}

// 选举超时时间，暂定为300ms，后续改为随机时间
func (r *Raft) electionTimeout() time.Duration {
	return time.Millisecond * 300
}

// 选举计时器
func (r *Raft) runElectionTimer() {
	electionDuration := r.electionTimeout()
	// 保存当前任期
	r.mu.Lock()
	savedTerm := r.currentTerm
	r.mu.Unlock()
	// 设置计时器
	ticker := time.NewTicker(10 * time.Millisecond)
	// 每隔10ms检测一次
	for {
		<-ticker.C
		// 操作临界变量上锁
		r.mu.Lock()
		// 如果已经完成选举，即不再是candidate
		if r.state != CandidateType && r.state != FollowerType {
			logrus.Infof("candidate %d become a leader, quit election", r.id)
			r.mu.Unlock()
			return
		}
		// 如果在选举时收到term更大leader心跳，会退回follower
		if r.currentTerm > savedTerm {
			logrus.Infof("candidate %d term changed from %d to %d, quit election", r.id, savedTerm, r.currentTerm)
			r.mu.Unlock()
			return
		}
		// 如果选举超时
		if dur := time.Since(r.electionTime); dur >= electionDuration {
			logrus.Infof("candidate %d election expired", r.id)
			// 走到这里说明没有选出新leader，开启下一轮选举
			r.startElection()
			r.mu.Unlock()
			return
		}
		r.mu.Unlock()
	}
}

// Leader逻辑
func (r *Raft) startLeader() {
	// 变更state
	r.state = LeaderType
	logrus.Infof("becomes Leader; term = %d, log = %v", r.currentTerm, r.log)

	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		// 发送心跳
		for {
			r.leaderSendHeartbeats()
			<-ticker.C

			r.mu.Lock()
			if r.state != LeaderType {
				r.mu.Unlock()
				return
			}
			r.mu.Unlock()
		}
	}()
}

func (r *Raft) leaderSendHeartbeats() {
	r.mu.Lock()
	savedCurrentTerm := r.currentTerm
	r.mu.Unlock()

	// 向所有节点发送心跳
	for _, peer := range r.peers {
		args := AppendEntriesArgs{
			Term:     savedCurrentTerm,
			LeaderID: r.id,
		}
		go func(peer int) {
			logrus.Infof("sending AppendEntries to %v: ni = %d, args = %+v", peer, 0, args)
			var reply AppendEntriesReply
			if ok := call(peer, "ConsensusModule.AppendEntries", args, &reply); !ok {
				return
			}
			r.mu.Lock()
			defer r.mu.Unlock()
			// 如果返回的任期更大，说明有更新的Leader，应当退为Follower
			if reply.Term > savedCurrentTerm {
				logrus.Infof("term out of date in heartbeat reply")
				r.startFollower(reply.Term)
				return
			}
		}(peer)
	}
}

// Follower逻辑
func (r *Raft) startFollower(term int) {
	logrus.Infof("becomes Follower with term=%d; log=%v", term, r.log)
	// 变更状态
	r.state = FollowerType
	// 设置term
	r.currentTerm = term
	r.voteFor = -1
	// 重置心跳时间
	r.electionTime = time.Now()

	go r.runElectionTimer()
}

// candidate发送选票以及处理选票
// pre: Leader心跳计时器超时，Follower变为candidate，向其他peer发送RequestVote
func (r *Raft) startElection() {
	// 变更角色为Candidate
	r.state = CandidateType
	// 自己的term ++
	r.currentTerm++
	savedTerm := r.currentTerm
	// 给自己投一票
	r.voteFor = r.id
	voteCnt := 1
	// 记录选举开始时间
	r.electionTime = time.Now()
	// 获取lastLog
	lastLog := r.log[len(r.log)-1]

	logrus.Infof("candidate %d starts election with term %d", r.id, savedTerm)

	// 循环，对每个peer进行RPC发送RequestVoteArgs
	for _, peer := range r.peers {
		go func(peer int) {
			// 封装args
			args := &RequestVoteArgs{
				Term:         r.currentTerm,
				CandidateID:  r.id,
				LastLogTerm:  lastLog.term,
				LastLogIndex: lastLog.index,
			}
			reply := &RequestVoteReply{}
			// 调用失败则返回
			if ok := call(peer, "Raft.RequestVote", args, reply); !ok {
				return
			}
			// 处理临街变量加锁
			r.mu.Lock()
			defer r.mu.Unlock()
			// 在处理期间其他rpc可能已经成功更新了节点state
			if r.state != CandidateType {
				logrus.Info("while waiting for reply, state = ", r.state)
				return
			}
			// 判断投票返回的信息
			// 如果任期比自己大，说明出现新的Leader，自动退回Follower
			if reply.Term > savedTerm {
				logrus.Info("Term out of date, a leader may already exist")
				r.startFollower(reply.Term)
			} else if reply.Term == savedTerm {
				// 没同意就返回
				if !reply.VoteGranted {
					return
				}
				// 同意则计票，判断是否到达法定人数
				voteCnt++
				if r.isQuorum(voteCnt) {
					logrus.Infof("candidate %d wins the election in term %d with %d votes", r.id, savedTerm, voteCnt)
					r.startLeader()
					return
				}
			}
		}(peer)
	}

	// 启动选举计时器
	go r.runElectionTimer()
}

// 收到投票请求后的处理
func (r *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) error {
	// 判断任期，任期小就拒绝
	if r.currentTerm > args.Term {
		reply.Term = r.currentTerm
		reply.VoteGranted = false
		return nil
	}
	// 如果收到了更大的任期，则通过
	if r.currentTerm < args.Term {
		// 更新term，记录投票
		r.currentTerm = args.Term
		r.voteFor = args.CandidateID
		// 通过
		reply.Term = r.currentTerm
		reply.VoteGranted = true
	}
	// 任期相同
	// 判断是否已经给同一个节点投过票了，如果投过返回true，因为candidate那边的统计是幂等的
	if r.voteFor == -1 || r.voteFor == args.CandidateID {
		lastLog := r.log[len(r.log)-1]
		// 判断candidate是否有足够新的日志
		// 不够新：args的term不够大或者term相同时index更小，拒绝
		if args.LastLogTerm < lastLog.term ||
			(args.LastLogTerm == lastLog.term && args.LastLogIndex < lastLog.index) {
			reply.Term = r.currentTerm
			reply.VoteGranted = false
			return nil
		}
		// 足够新则同意投票
		reply.Term = r.currentTerm
		reply.VoteGranted = true
		// 记录投票
		r.voteFor = args.CandidateID
	}
	return nil
}

func (r *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) error {
	return nil
}
