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

##
- 222行判断Term是多余的，只要PrevlogIndex有东西，就应该以Leader的为准
- 237行切错了，应该从0切到PrevlogIndex
- 369行删掉
