package raft

type RequestVoteArgs struct {
	// 自己当前的term
	Term int
	// 自己的ID
	CandidateID int
	// candidate最后一条日志的term
	LastLogTerm int
	// candidate最后一条日志的index
	LastLogIndex int
}

type RequestVoteReply struct {
	// 接到投票请求节点的任期，如果拉票者的任期落后，则通过这里的任期来更新
	Term int
	// 是否投票
	VoteGranted bool
}

type CallArgs struct {
	// 本次请求的token
	Token string
	// 要发送rpc目的节点的id
	Id      int
	Method  string
	Args    interface{}
	Reply   interface{}
	Success bool
}

type AppendEntriesArgs struct {
	// 当前term和ID
	Term     int
	LeaderID int

	// 前一条日志的index和term，用于日志与从节点不一致时的恢复
	PrevLogIndex int
	PrevLogTerm  int
	// 要同步的日志
	Entries []LogEntry
	// Leader已提交的最后一条日志，Follower可根据此参数判断自己本地哪些日志可以提交
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}
