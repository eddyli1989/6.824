## Test 2A Initial election ,3 peers

- Index2 首先变成Candidite，并且成功收到了投票，问题是收到投票后没有立即发起AppendEntries，而是去睡觉了 
-   - [ ] 需要在变成Leader后立即发送AppendEntries
- Index0 收到RequestVote以后，是否要打断自己的选举计时器?，因为RequestVote会更新自己的任期，此时如果与AppendEntries并发可能会导致重新选举一次
-   - [ ] 考虑在接受RequestVote时把RecvHB设置以下，让自己晚一轮变成Candidate，这样可以缩短选举时间
- 由于没有收到心跳，Index0 变成了Candidate，Term2 给Index 1和 Index 2 发送了RequestVote 
    - 由于Index 2还在休眠（此时还在加锁），所以Index 2收到RequestVote后，由于拿不到锁，卡住了
    -   - [ ] 需要减少Candidate的锁粒度，不要在休眠的时候加锁(排查下所有的锁粒度)
    - 由于Index 1也是Candidate，也存在上述的问题
    - 假如 Index 2 能够拿到锁，那么Term > currentTerm，能够接纳投票，但却没有设置自己的状态到Folowwer
    -    - [ ] 在接纳投票的时候把自己强制设置成Follwer
    - 假如 Index 1 拿到锁，但是Index 1 和 Index0的 Term相同，且Index1 已经投票给了自己，所以Index1不会给 Index0 投票
    - Index 1 最终拿到了 Index0的票变成新的Leader

## Test 2A Election after network failure, 3 peers:
- 这里打印出了上一个用例的日志，引起了混淆
-   - [ ] 需要在接收到消息的时候先看一下Killed
-   Index 0
