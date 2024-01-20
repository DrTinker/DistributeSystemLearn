package basicpaxos

import (
	"testing"
	"time"
)

func start(aids, lids []int) ([]*Acceptor, []*Learner) {
	// 启动acceptors
	Acceptors := make([]*Acceptor, len(aids))
	for i, aid := range aids {
		acceptor := NewAcceptor(aid, lids)
		acceptor.start()
		Acceptors[i] = acceptor
	}
	// 启动learners
	Learners := make([]*Learner, len(lids))
	for i, lid := range lids {
		learner := NewLearner(lid, aids)
		learner.start()
		Learners[i] = learner
	}

	return Acceptors, Learners
}

func close(acceptors []*Acceptor, leraners []*Learner) {
	for _, a := range acceptors {
		a.close()
	}

	for _, l := range leraners {
		l.close()
	}
}

func TestSingleProposer(t *testing.T) {
	// acceptors: 1001 1002 1003
	aids := []int{1001, 1002, 1003}
	// learner: 2001
	lids := []int{2001}

	// 启动acceptors和learners
	acceptors, learners := start(aids, lids)
	defer close(acceptors, learners)
	// time.Sleep(10 * time.Second)
	// 创建proposer: 3001
	proposer := NewProposer(3001, aids)
	// 执行提案
	res := proposer.Propose("value1")
	if res != "value1" {
		t.Errorf("proposer value = %v expected %v\n", res, "value1")
		return
	}
	// 查看learner最终的值
	// 形成共识后再请求
	time.Sleep(1 * time.Second)
	learnRes := learners[0].chosen()
	if learnRes != "value1" {
		t.Errorf("learner value = %v expected %v\n", learnRes, "value1")
		return
	}

	t.Logf("Success\n proposer value = %v learner value = %v\n", res, learnRes)
}

func TestTwoProposer(t *testing.T) {
	// acceptors: 1001 1002 1003
	aids := []int{1001, 1002, 1003}
	// learner: 2001
	lids := []int{2001}

	// 启动acceptors和learners
	acceptors, learners := start(aids, lids)
	defer close(acceptors, learners)

	// 创建proposer1: 3001
	proposer1 := NewProposer(3001, aids)
	// 执行提案
	res1 := proposer1.Propose("value1")
	// 创建proposer2: 3002
	proposer2 := NewProposer(3002, aids)
	// 执行提案
	res2 := proposer2.Propose("value2")
	if res1 != res2 {
		t.Errorf("proposer value1 = %v value2 = %v\n", res1, res2)
	}
	// 查看learner最终的值
	// 形成共识后再请求
	time.Sleep(1 * time.Second)
	learnRes := learners[0].chosen()
	if learnRes != res1 {
		t.Errorf("learner value = %v expected %v\n", learnRes, res1)
	}

	t.Logf("Success\n proposer value = %v learner value = %v\n", res1, learnRes)
}
