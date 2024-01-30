# Raft的实现

### 三种角色

- Leader：只能有一个
- Candidate：Leader心跳超时后，Follower自动转化为Candidate
- Follower：从节点



### 基础数据结构

```go
// 一个Raft节点的数据结构
type Raft struct {
    // 标识本节点当前身份
    state int
    // term + index 唯一定位一条日志
    currentTerm int
    log []*LogEntry
    // 记录投票信息
    voteFor int
    // 心跳超时计时器
    heartBeatTime
}

// 一个Log的结构，通过 term + index 唯一标识一条日志
type LogEntry struct {
    term int
    index int
    value interface{}
}
```



### 领导者选举

当节点的心跳计时超时后，会自动升级为Candidate，将自己当前任期加一，并向其他节点发送投票请求(RequestVote)

```go
type RequestVoteArgs struct {
    // 自己当前的term
    term int
    // 自己的ID
    candidateID int
}

type RequestVoteReply struct {
    // 接到投票请求节点的任期，如果拉票者的任期落后，则通过这里的任期来更新
    term int
    // 是否投票
    VoteGranted int
}
```

当一个节点接到投票请求，根据以下两条原则判断是否通过

- 任期是否比自己大：判断自己和请求中的term
- 日志是否足够新：

当拉票者收到超过半数的同意后就自动升级为Leader，并开始向所有节点发送心跳，收到心跳的节点会从Candidate转换成Follower



### 日志复制





### etcd

算法层：调用send -> 结果写入Message -> raft.advance把Message写入Ready -> Ready传入readyc

应用层：从readyc中取出Ready -> 解析Message执行操作