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
- - [x] 在收到RequestVote以后，除了校验Term之外还得看谁的日志长
- Index0 收到AppendLog以后把Index1和Index2的日志全清Test 


## Test 2B Test (2B): agreement after reconect
— - [x] Index1在重新进行选举的时候，由于Index0是follower,此时VoteFor的取值是1，新一轮开始以后index1请求投票，竟然被0拒绝了
- - [x] 在重试Log同步时，leader commit字段也得重新赋值
- - [x] 重试同步log成功时，nextIndex是老的，没有更新
- - [x] Match算的是错的
- - [x] Success Count变化后应该立即Commit
- - [x] Commit Index的计算需要根据Log长度和match数组进行计算，不然有可能有的日志永远不会被提交。

## TestFailNoAgree2B
- - [x] 每个人都投给自己，结果无法选出Leader，每个人每一轮只能投一个人，因此VoteFor是和Term绑定的，如果Term变了，那么VoteFor应该变为无效
- - [x] log在重试同步的时候似乎忘记填Term字段了

## Test (2B): rejoin of partitioned leader ...
- - [x] 发完消息后判断一下kill，如果kill了就退出
- - [x] 3个Peer，Index0 Leader，日志长度5，然后0和1，2分区，1，2中，2短暂变为Leader并且接受了一个日志同步到了1，后来和0联通以后，0同步自己的日志到1，2，此时1，2的日志位置2已经和0不同且已经Apply，再次Apply后将报错
- - [x] 判断日志是否更新的逻辑是错的，首先应该先看最后一个日志的Term，哪个大哪个更新，在Term相等的情况下，才会去看日志长度

- 主要过程： peers:3个，先来1个101，达成共识，然后断开leader，然后给leader上3个日志102，103，104，显然无法提交
- 然后另外两个peers选一个leader出来，上1个103，此时日志已经分裂
- 然后leader2断开
- 阶段1：Index2当选，来1个101，同步到0和1，此时日志长度1，leaderCommit=1,Term=1
- 阶段2: Index2断开，Index0当选，Term=2.Index2来到102，103，104，Index2日志长度4
- 阶段3：Index0收到103，同步到Index1,此时Index0,1的日志为[101,103],LeaderCommit=1,Index2日志为[101,102,103,104],
- 阶段4：Index0断开，Index2连上，Index2：又收到了104，但是由于Term小，Index2变成了Folowwer,Index1变为新Leader,Term=3
    Index2的日志被1刷成了[101,103]，注意这里103日志的Term=？.(前面任期的日志如何刷)
- 阶段5：index0重新加入，最后来一个105
## Test (2B): leader backs up quickly over incorrect follower logs ...
- 阶段1：在线（01234），长度均为1，Term1,leader:3
  5个Peer (0,1,2,3,4), index 3 当选，来了一个日志5347949693413728018，成功提交，leaderCommit=1, 

- 阶段2：断开0,1,2，在线（3,4）,leader:3,log len:3,4:51,0，1,2:1，LeaderCommit=1
 index3收到50个命令，均未提交,index3给index4同步了50个日志，此时index3,4分别有51个日志，
 
- 阶段3：断开3,4，在线（0,1,2），leader:0，log len 0,1,2:51,LeaderCommit=51，Term2
 此时index3,4断开，0，1,2重新加入，0变为Leader，并把自己唯一的日志同步到了1和2，然后0收到了50个日志，成功同步给1，2；此时012一共51个日志且都成功提交

- 阶段4:断开1，3,4，在线（0,2），leader:0,log len:0,2:101, 1:51,3,4:51 ，Term2
 0再次收到50个日志，这50个应该都不能提交；此时0,2一共有101个日志
 
- 阶段5：断开（0,2），在线（3,4,1），leader:3，Term1比较小，1应该当选新的Leader
 然后全部断开，连上3,4,1，再次提交50个
 - [ ] 3为啥有52个日志???
 - [ ] 1由于一直醒的比较晚，因此一直没有当选Leader
 - [ ] 随机数范围搞大一点，搞一个比较真的随机数试试
 - [ ] 日志同步效率太低了，在瞬间收到大量日志时，每次都起新的携程同步会耗费大量资源且不可控制
 
 阶段6：在线（0,1,2,3,4）
 然后全部连上，在提交一个
