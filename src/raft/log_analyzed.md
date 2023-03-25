## Test 2A Initial election ,3 peers

- Index2 首先变成Candidite，并且成功收到了投票，问题是收到投票后没有立即发起AppendEntries，而是去睡觉了 
-   - [x] 需要在变成Leader后立即发送AppendEntries
- Index0 收到RequestVote以后，是否要打断自己的选举计时器?，因为RequestVote会更新自己的任期，此时如果与AppendEntries并发可能会导致重新选举一次
-   - [x] 考虑在接受RequestVote时把RecvHB设置以下，让自己晚一轮变成Candidate，这样可以缩短选举时间
- 由于没有收到心跳，Index0 变成了Candidate，Term2 给Index 1和 Index 2 发送了RequestVote 
    - 由于Index 2还在休眠（此时还在加锁），所以Index 2收到RequestVote后，由于拿不到锁，卡住了
    -   - [x] 需要减少Candidate的锁粒度，不要在休眠的时候加锁(排查下所有的锁粒度)
    - 由于Index 1也是Candidate，也存在上述的问题
    - 假如 Index 2 能够拿到锁，那么Term > currentTerm，能够接纳投票，但却没有设置自己的状态到Folowwer
    -    - [x] 在接纳投票的时候把自己强制设置成Follwer
    - 假如 Index 1 拿到锁，但是Index 1 和 Index0的 Term相同，且Index1 已经投票给了自己，所以Index1不会给 Index0 投票
    - Index 1 最终拿到了 Index0的票变成新的Leader

## Test 2A Election after network failure, 3 peers:
- 这里打印出了上一个用例的日志，引起了混淆
-   - [ ] 需要在接收到消息的时候先看一下Killed
-   Index 0

## Test 2A Passed

## Test 2B 1

- 收到AppendEntries时，需要判断PrevLogIndex，如果PrevLogIndex == 0 ,代表是第一条日志，此时应该无条件接受并覆盖自己的日志，如果<0那么非法返回，如果>0那么检查PrevLogIndex-1的Term与PrevLogTerm是否相等，如果不等于那么return false，否则接纳该消息并更新日志
- appendReq := rf.getAppendEntrisArg()，这个应该放到for循环中，不然各个peers之间会互相影响
- 考虑把sendAppendLogAsync改成同步函数，然后在外部改成go func()的调用方式
- 
## Test 2B Test (2B): basic agreement 2
- Index0 竞选胜利，开始同步日志，但是没有过滤自己，导致自己收到了自己同步的日志并且把自己改成了Folowwer
- Index2 重新竞选变成主，开始发送心跳，由于在接受心跳的时候没有判断是否有内容，结果导致所有人把日志清理掉了
- 之后就一直发送心跳

## est 2B Test (2B): basic agreement 2
- 222行判断Term是多余的，只要PrevlogIndex有东西，就应该以Leader的为准
- 237行切错了，应该从0切到PrevlogIndex
- 369行删掉

## Test 2B Test (2B): agreement after reconect
- Index2 Leader，接受到append，同步到0和1成功，并且都应用到自动机
- Index2 再次收到了两次appendLog，但是这次只同步到了1，0此时被割裂了，但是因为日志提交成功大于N/2，所以该日志也被提交到自动机
- 此时log len为3，commitIndex=3,lastApplid=2
- Index0 被割裂，变为Candidate
- Index2 再次收到AppendLog 此时 LogLen=4,同步到1成功，同步到0失败
- Index2 再次收到AppendLog 两次，此时LogLen=6，此时0上线，收到Append消息，PrevLogIndex=5
- 由于0在被割裂以后变成了Candidate，因此Term比较大，回复的Term=2大于1，因此Index2直接变为Folowwer了
- Index0 重新变为Candidate，Term 4，由于Term更大Index0变成Leader了！！！，但是Index0的Log长度是1！！！
- [] 在收到RequestVote以后，除了校验Term之外还得看谁的日志长
- Index0 收到AppendLog以后把Index1和Index2的日志全清Test 


2B Test (2B): agreement after reconect

— Index1在重新进行选举的时候，由于Index0是follower,此时VoteFor的取值是1，新一轮开始以后index1请求投票，竟然被0拒绝了
- 在重试Log同步时，leader commit字段也得重新赋值
