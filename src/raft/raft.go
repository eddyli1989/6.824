package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"
	"fmt"
	"sync"
	"sync/atomic"

	//	"6.824/labgob"
	"6.824/labrpc"

	"math/rand"
	"time"
)

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

const (
	LEADER = iota
	CANDIDATE
	FOLLOWER
)

const ElectionBaseTime = 500 * time.Millisecond
const ElectionRandTime = 100 // ms
const HBInterval = 150 * time.Millisecond

type LogEntries struct {
	Term    int
	Index   int
	Content interface{}
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	applyCh chan ApplyMsg

	// persistent state
	currentTerm int
	voteFor     int
	log         []LogEntries

	// Volatile state 4 log
	commitIndex int
	lastApplied int

	// leaders state
	nextIndex  []int
	matchIndex []int

	// added
	role          atomic.Int32 // my role
	currentLeader int          // currentLeader index
	voteNum       int          // voteNum i got
	rcvdHB        atomic.Bool  // rcvdHB or Not
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	// Your code here (2A).
	DPrintf("Index:%d, role:%d GetState", rf.me, rf.role.Load())
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.currentTerm, rf.role.Load() == LEADER
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {

	// Your code here (2D).

	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandiddateID int
	LastLogIndex int
	LastLogTerm  int
}

type AppendEntriesArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntries
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool // accept or not
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	DPrintf("Index:%d, role:%d rcvd AppendEntries,from:%d, log len:%d, Prev:%d, leaderCommit:%d", rf.me, rf.role.Load(), args.LeaderId, len(args.Entries), args.PrevLogIndex, args.LeaderCommit)
	if args.Term < rf.currentTerm {
		DPrintf("Index:%d, Reject AppendEntries term:%d is small", rf.me, args.Term)
		reply.Success = false
		reply.Term = rf.currentTerm
		return
	}

	if args.PrevLogIndex < 0 {
		DPrintf("Index:%d, Reject AppendEntries PrevLogIndex:%d is invalid", rf.me, args.PrevLogIndex)
		return
	}

	if args.PrevLogIndex != 0 {
		if len(rf.log) < args.PrevLogIndex {
			DPrintf("Index:%d, Reject AppendEntries log length :%d is small than PrevLogindex:%d", rf.me, len(rf.log), args.PrevLogIndex)
			reply.Success = false
			reply.Term = rf.currentTerm
			return
		}

		if rf.log[args.PrevLogIndex-1].Term != args.PrevLogTerm {
			DPrintf("Index:%d, Reject AppendEntries PrevLogTerm is not equal :%d vs :%d,index:%d", rf.me, rf.log[args.PrevLogIndex-1].Term, args.PrevLogTerm, args.PrevLogIndex-1)
			reply.Success = false
			reply.Term = rf.currentTerm
			return
		}
	}

	if len(args.Entries) > 0 {
		if args.PrevLogIndex == 0 {
			DPrintf("Index:%d, copy all log from:%d", rf.me, args.LeaderId)
			rf.log = make([]LogEntries, len(args.Entries))
			copy(rf.log, args.Entries)
		} else {
			DPrintf("Index:%d, copy %d log , start point:%d, from:%d", rf.me, len(args.Entries), args.PrevLogIndex, args.LeaderId)
			if len(rf.log) > args.PrevLogIndex {
				DPrintf("Index:%d, log len:%d is bigger than PrevLogIndex:%d, cut", rf.me, len(rf.log), args.PrevLogIndex)
				rf.log = rf.log[:args.PrevLogIndex]
			}
			rf.log = append(rf.log, args.Entries...)
		}
	}
	if args.LeaderCommit > rf.commitIndex {
		DPrintf("Index:%d, update commit index :%d, leaderCommit:%d", rf.me, rf.commitIndex, args.LeaderCommit)
		if args.LeaderCommit > len(rf.log) {
			rf.commitIndex = len(rf.log)
		} else {
			rf.commitIndex = args.LeaderCommit
		}
	}
	rf.apply()
	rf.currentTerm = args.Term
	rf.rcvdHB.Store(true)
	rf.changeRole(FOLLOWER)
	rf.currentLeader = args.LeaderId

	reply.Success = true
	reply.Term = rf.currentTerm
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	DPrintf("Index:%d, role:%d rcvd RequestVote from:%d", rf.me, rf.role.Load(), args.CandiddateID)
	reply.Term = rf.currentTerm

	if args.Term < rf.currentTerm {
		DPrintf("Index:%d, Reject vote to:%d Req term:%d, currentTerm:%d", rf.me, args.CandiddateID, args.Term, rf.currentTerm)
		reply.VoteGranted = false
		return
	}

	if args.LastLogIndex < len(rf.log) {
		DPrintf("Index:%d Reject vote to:%d LastLogIndex :%d is smmal than me:%d", rf.me, args.CandiddateID, args.LastLogIndex, len(rf.log))
		reply.VoteGranted = false
		return
	}

	if args.LastLogIndex > 0 && len(rf.log) == args.LastLogIndex && rf.log[args.LastLogIndex-1].Term > args.LastLogTerm {
		DPrintf("Index:%d Reject vote to:%d LastLogTerm :%d is small than me:%d", rf.me, args.CandiddateID, args.LastLogTerm, rf.log[args.LastLogIndex-1].Term)
		reply.VoteGranted = false
		return
	}

	// if rf.role.Load() != FOLLOWER {
	// 	DPrintf("Index:%d Reject vote to:%d same term but i'm not follower, role:%d", rf.me, args.CandiddateID, rf.role.Load())
	// 	reply.VoteGranted = false
	// 	return
	// }

	if rf.voteFor != -1 {
		DPrintf("Index:%d, Reject vote to:%d, already voteFor:%d, term:%d", rf.me, args.CandiddateID, rf.voteFor, rf.currentTerm)
		reply.VoteGranted = false
		return
	}

	// todo check log index
	DPrintf("Index:%d, Accept vote to:%d, term:%d, current term:%d", rf.me, args.CandiddateID, args.Term, rf.currentTerm)

	rf.changeRole(FOLLOWER)
	reply.VoteGranted = true

	rf.voteFor = args.CandiddateID
	rf.currentTerm = args.Term

	rf.rcvdHB.Store(true)
}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) sendAppendLogAsync(server int, args *AppendEntriesArgs, ch chan bool) {
	go func(index int) {
		var appendRsp AppendEntriesReply
		var failedCount int

		for !rf.killed() {
			ret := rf.sendAppendEntries(index, args, &appendRsp)
			if ret {
				rf.mu.Lock()
				DPrintf("Index:%d Send AppendEntries to :%d Successed", rf.me, index)
				if appendRsp.Term > rf.currentTerm {
					rf.currentTerm = appendRsp.Term
					rf.changeRole(FOLLOWER)
					rf.mu.Unlock()
					ch <- false
					return
				}

				// if log not match, try again with minus PrevLogIndex
				if !appendRsp.Success {
					DPrintf("Index:%d Send AppendEntries to :%d log not match, failedCount:%d", rf.me, index, failedCount)
					failedCount++
					args.PrevLogIndex -= failedCount
					if args.PrevLogIndex < 0 {
						args.PrevLogIndex = 0
					}

					if args.PrevLogIndex > 0 {
						args.PrevLogTerm = rf.log[args.PrevLogIndex-1].Term
					}

					args.Entries = rf.log[args.PrevLogIndex:]
					DPrintf("Index:%d change nextIndex:%d from %d to %d", rf.me, server, rf.nextIndex[server], args.PrevLogIndex+1)
					rf.nextIndex[server] = args.PrevLogIndex + 1
					rf.mu.Unlock()
					continue
				}

				rf.matchIndex[server] += len(args.Entries)
				DPrintf("Index:%d Sync log to :%d Successed, next:%d,match:%d,len:%d", rf.me, index, rf.nextIndex[server], rf.matchIndex[server], len(args.Entries))

				// success
				rf.mu.Unlock()
				ch <- true
				return
			}
			// network failed
			ch <- false
			break
		}
		DPrintf("Index:%d Send AppendEntries to :%d Failed", rf.me, index)
	}(server)
}

func (rf *Raft) appendLog(command interface{}) (int, int) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	log2Append := LogEntries{Term: rf.currentTerm, Index: len(rf.log) + 1, Content: command}
	rf.log = append(rf.log, log2Append)
	DPrintf("Index:%d, appendLog index:%d , term:%d ,log len:%d", rf.me, len(rf.log), rf.currentTerm, len(rf.log))
	return len(rf.log), rf.currentTerm
}

func (rf *Raft) apply() {
	for i := rf.lastApplied; i < rf.commitIndex; i++ {
		DPrintf("Index:%d, apply msg :%d, role:%d", rf.me, i, rf.role.Load())
		var msg ApplyMsg
		msg.Command = rf.log[i].Content
		msg.CommandValid = true
		msg.CommandIndex = rf.log[i].Index
		rf.applyCh <- msg
		rf.lastApplied++
	}
}

func (rf *Raft) syncLog(index int) {
	ch := make(chan bool, len(rf.peers))
	for i := 0; i < len(rf.peers); i++ {
		if i == rf.me {
			continue
		}
		rf.mu.Lock()
		appendReq := rf.getAppendEntrisArg()
		appendReq.PrevLogIndex = rf.nextIndex[i] - 1
		if appendReq.PrevLogIndex > len(rf.log) || appendReq.PrevLogIndex < 0 {
			panic(fmt.Sprintf("Invalid prev log index:%d, log length:%d", appendReq.PrevLogIndex, len(rf.log)))
		}
		if appendReq.PrevLogIndex > 0 {
			appendReq.PrevLogTerm = rf.log[appendReq.PrevLogIndex-1].Term
		}
		appendReq.Entries = rf.log[appendReq.PrevLogIndex:]
		DPrintf("Index:%d Update next index:%d from %d to %d", rf.me, i, rf.nextIndex[i], len(rf.log)+1)
		rf.nextIndex[i] = len(rf.log) + 1
		rf.mu.Unlock()
		rf.sendAppendLogAsync(i, &appendReq, ch)
	}
	successCount := 1
	totalCount := 0
	for successCount <= len(rf.peers)/2 && totalCount < len(rf.peers)-1 {
		ret := <-ch
		if rf.killed() || rf.role.Load() != LEADER {
			return
		}
		if ret {
			successCount++
		}
		totalCount++
	}

	if successCount > len(rf.peers)/2 {
		DPrintf("Index:%d Inc commit index:%d", rf.me, rf.commitIndex)
		rf.commitIndex++
		rf.mu.Lock()
		rf.apply()
		rf.mu.Unlock()
	}
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := rf.role.Load() == LEADER
	DPrintf("Index:%d Recv Start, rold:%d", rf.me, rf.role.Load())
	if !isLeader {
		return index, term, isLeader
	}

	// Your code here (2B).
	index, term = rf.appendLog(command)

	go rf.syncLog(index)
	return index, term, isLeader
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	DPrintf("Index:%d, kill", rf.me)
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) getRandomTicker(base time.Duration) time.Duration {
	return base + (time.Duration)(rand.Intn(ElectionRandTime))*time.Millisecond
}

func (rf *Raft) initLeaderData() {
	rf.nextIndex = make([]int, 0, len(rf.peers))
	rf.matchIndex = make([]int, 0, len(rf.peers))
	for i := 0; i < len(rf.peers); i++ {
		rf.nextIndex = append(rf.nextIndex, len(rf.log)+1)
		rf.matchIndex = append(rf.matchIndex, 0)
	}
}

func (rf *Raft) changeRole(role int) {
	DPrintf("Index:%d, change role from:%d to :%d", rf.me, rf.role.Load(), role)
	if role == FOLLOWER && rf.role.Load() != FOLLOWER {
		rf.voteFor = -1
		rf.rcvdHB.Store(false)
	}
	if role == CANDIDATE {
		rf.voteNum = 0
	}
	if role == LEADER {
		rf.currentLeader = rf.me
	}
	rf.role.Store((int32)(role))
}

func (rf *Raft) processLeader() {
	if rf.role.Load() != LEADER {
		return
	}
	DPrintf("Index:%d, change to leader", rf.me)
	ch := make(chan bool, len(rf.peers))
	for j := 0; j < len(rf.peers); j++ {
		if j == rf.me {
			continue
		}
		if rf.role.Load() != LEADER {
			return
		}
		appendReq := rf.getAppendEntrisArg()
		var appendRsp AppendEntriesReply
		// todo:check log
		DPrintf("Index:%d Try Send AppendEntries to :%d", rf.me, j)

		go func(index int) {
			defer func() {
				ch <- false
				DPrintf("Index:%d AppendEntries to :%d  finish", rf.me, index)
			}()
			ret := rf.sendAppendEntries(index, &appendReq, &appendRsp)
			if ret {
				rf.mu.Lock()
				defer rf.mu.Unlock()
				DPrintf("Index:%d Send AppendEntries to :%d Sucsicced", rf.me, index)
				if appendRsp.Term > rf.currentTerm {
					rf.currentTerm = appendRsp.Term
					rf.changeRole(FOLLOWER)
					rf.initLeaderData()
					ch <- true
				}
				return
			}
			DPrintf("Index:%d Send AppendEntries to :%d Failed", rf.me, index)
		}(j)
	}

	var count int = 0
	var sleeped time.Duration = 0
	var waitTime = 5 * time.Millisecond
	for count != (len(rf.peers)-1) && sleeped < HBInterval {
		select {
		case ret := <-ch:
			count++
			// role changed
			if ret {
				DPrintf("Index:%d Out loop in processLeader, sleep:%d ms", rf.me, sleeped/time.Millisecond)
				return
			}
		case <-time.After(waitTime):
			sleeped += waitTime
		}
	}

	DPrintf("Index:%d Out loop in processLeader, sleep:%d ms", rf.me, sleeped/time.Millisecond)
	if sleeped < HBInterval {
		time.Sleep(HBInterval - sleeped)
	}
}

func (rf *Raft) getAppendEntrisArg() AppendEntriesArgs {
	var appendReq AppendEntriesArgs
	appendReq.LeaderCommit = rf.commitIndex
	appendReq.LeaderId = rf.me
	appendReq.Term = rf.currentTerm

	appendReq.PrevLogIndex = 0             // todo:
	appendReq.PrevLogTerm = rf.currentTerm // todo:

	return appendReq
}

func (rf *Raft) getVoteArgs() RequestVoteArgs {
	var req RequestVoteArgs
	req.Term = rf.currentTerm
	req.CandiddateID = rf.me
	req.LastLogIndex = len(rf.log)
	if len(rf.log) > 0 {
		req.LastLogTerm = rf.log[len(rf.log)-1].Term
	} else {
		req.LastLogTerm = rf.currentTerm
	}
	return req
}

func (rf *Raft) processCandidate() {
	rf.mu.Lock()
	rf.currentTerm++
	DPrintf("Index:%d, change to candidate, term:%d", rf.me, rf.currentTerm)
	rf.voteFor = rf.me
	rf.voteNum = 1
	rf.mu.Unlock()

	reqVote := rf.getVoteArgs()
	ch := make(chan bool, len(rf.peers))
	for i := 0; i < len(rf.peers); i++ {
		if i == rf.me {
			continue
		}
		if rf.role.Load() != CANDIDATE {
			return
		}

		go func(index int) {
			defer func() {
				ch <- false
				DPrintf("Index:%d RequestVote to :%d  finish", rf.me, index)
			}()
			var rsp RequestVoteReply
			DPrintf("Index:%d, send RequestVote to :%d", rf.me, index)
			ret := rf.sendRequestVote(index, &reqVote, &rsp)
			if !ret {
				DPrintf("Index:%d, send RequestVote to :%d FAILED!", rf.me, index)
				return
			}

			rf.mu.Lock()
			defer rf.mu.Unlock()
			DPrintf("Index:%d, send RequestVote to :%d Success", rf.me, index)
			if rsp.Term > rf.currentTerm {
				rf.currentTerm = rsp.Term
				rf.changeRole(FOLLOWER) // how to return ?
				ch <- true
				return
			}
			if rsp.VoteGranted {
				rf.voteNum++
				if rf.voteNum > len(rf.peers)/2 {
					rf.changeRole(LEADER) // how to return ?
					ch <- true
					return
				}
			}
		}(i)
	}
	count := 0
	var sleeped time.Duration = 0
	var waitTime = 5 * time.Millisecond
	for count != (len(rf.peers)-1) && sleeped < ElectionBaseTime {
		select {
		case ret := <-ch:
			count++
			if ret {
				DPrintf("Index:%d Out loop in processCandidate, sleep:%d ms", rf.me, sleeped/time.Millisecond)
				return
			}
		case <-time.After(waitTime):
			sleeped += (waitTime)
			continue
		}
	}
	DPrintf("Index:%d Out loop in processCandidate, sleep:%d ms", rf.me, sleeped/time.Millisecond)
	if sleeped > ElectionBaseTime {
		return
	}

	time.Sleep(rf.getRandomTicker(ElectionBaseTime - sleeped))
}

func (rf *Raft) processFollwer() {
	if rf.role.Load() != FOLLOWER {
		return
	}
	time.Sleep(rf.getRandomTicker(ElectionBaseTime))
	DPrintf("Index:%d, change to follower", rf.me)
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.currentLeader == -1 || !rf.rcvdHB.Load() {
		rf.changeRole(CANDIDATE)
		return
	}
	rf.rcvdHB.Store(false)
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	for !rf.killed() {

		// Your code here to check if a leader election should
		// be started and to randomize sleeping time using
		// time.Sleep().

		DPrintf("Index:%d, Role:%d, In loop, log len:%d", rf.me, rf.role.Load(), len(rf.log))
		switch rf.role.Load() {
		case FOLLOWER:
			rf.processFollwer()
		case CANDIDATE:
			rf.processCandidate()
		case LEADER:
			rf.processLeader()
		}
	}
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.currentTerm = 0
	rf.role.Store(FOLLOWER)
	rf.currentLeader = -1
	rf.voteFor = -1

	rf.log = make([]LogEntries, 0, 10)
	rf.initLeaderData()
	rf.applyCh = applyCh

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	DPrintf("Index:%d, we have %d peers", rf.me, len(peers))
	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}
