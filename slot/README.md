# Slot
Simple decentralized consensus using reasonably synced clocks and VRFs

## Assumptions
Consider: Reasonably synced clocked across the network (NTP, +/-50ms). Or an
implementation of something like:
 - https://arxiv.org/pdf/1707.08216.pdf
 - https://www.distributed-systems.net/my-data/papers/2006.selfman.pdf
Or a subset of the network is picked to perform a PoW to act as as clock? Or everyone
adjusts based on network time of seen blocks. Or a setup like:
https://github.com/chainpoint/whitepaper/issues/6

## TODO
- [x] find a tip function that is adaptive to blocks arriving while a tip was chosen
     that is then lowered in rank. Start at the highest round and go back until the
     difference between the strongest block is max size of 256bits or the genesis block
     is reached. Not a problem because 1) we only add tips, no the new block is always
     compared and 2) we can use dynamic strength calculation to measure after block was
     added.
- [] find a threshold function that uses that finds a moving average over the last
     N block as indication of how good the proposers are at finding high values, a
     large network should be able to find super high values. Some high draws might
     be lucky, but overall it should indicate the size of the network.
- [x] implement the out of order message handler
- [x] implement the broadcast deduplication filter
- [x] find a way to resolve blocks from the handler functions
- [x] wireframe the proposal handling function
- [x] design how the engine starts up from midway or if there are no messages
- [x] figure out how to trigger the voter releasing its votes:
      - have a channel that can be used to trigger it manually (for testing)
      - should allow for triggering after a certain blockTime (from the engine)
      - should only trigger once, afterwards all proposals that are higher should
        broadcast as votes right away
- [x] test first system logic
  - [x] test block proposal from genesis
  - [x] test proposal counting by voter
  - [x] test vote broadcast before block time
  - [x] test vote broadcast after block time
  - [x] test vote counting and appending
  - [x] test new tip and new proposal (and the cycle continues)
  - [x] test the cycle with 2 members
  -   [x] add some debug utilities. label members and blocks in logs (compile flag?)
  -   [x] debug deadlock where one vote gets lost amongst 2 members with messages only
          flowing after blocktime has expired
  - [x] create a telemetry wrapper for broadcast for metrics
  - [x] create utility function for named PKs and Blocks
  - [x] fix deadlock for 0 minVotes scenario
  - [ ] draw chain shape to look at shape
  - [ ] assert consensus
  - [ ] fix race detections
  - [ ] fix deadlock when using race detector
  - [x] stopping voting in a round should not be done after a single vote, but
        after the majority has come in
  - [ ] build a testing method against our in-memory broadcast setup
   - [x] add collect method
   - [ ] add latency simulation?
  - [ ] setup a test with large amount of engines
   - [ ] test ooo on large scale
   - [ ] reproduce minimal error with vrf sigs failing
  - [ ] work on the threshold functions for: Voters, Proposer and MinVotes
  - [ ] theorize: how to resume after deadlock?
  - [ ] fix on split tip voter problem:
    - [x] start with a fixed seed per round, role shouldn't change if a new tip
          comes along at the same height. Still locks up the protocol sometimes
    - [ ] can we find a common seed even with multiple tips? Biggest common subgraph of strongest tips?


## Limitations/Problems

- [ ] Problem 1: figure out to adjust drawing difficulty: look at the average draw of leaders
      at the last N rounds and extrapolate how many nodes would be needed to draw
      these results:
- [ ] Problem 2: What prevents a large portion of the voters to only vote on blocks
      that would cause them to be proposers the next round? Not a big issue? required
      majority of the network as they need to control both majority of voters and
      majority of proposers and they can't choose the seed anyway.
- [ ] Problem 3: what happens if there are enough proposers to reach the threshold
      value? This may happen if the threshold dips too low or when the network is
      segmented?
- [ ] Problem 4: figure out how to protect against grinding attack that tries out many old
      blocks as 'prev' to find a very high draw: If voters take into account the full chain
      strength they should never vote on old ref blocks.
- [ ] Problem 5: Voter privileges are tied to the current tip a member is working, if it
      switches it will try to vote for blocks from another tip. This will not verify correctly
      for by members. Causing a deadlock sometimes. See exp1 for a way to reproduce this.
      Possible solution: draw a ticket for every tip. Propose and vote for each?
- [ ] Problem 6: Do voters have upfront knowledge of which block will be highest?
- [ ] Problem 7: Why would proposers wait for all votes before proposing? Should include
      some proof of seeing votes from N notaries


## Idea: Time-lock puzzle proof
- https://crypto.stackexchange.com/questions/9327/parallel-resistant-proof-of-work-scheme
- http://www.hashcash.org/papers/time-lock.pdf

## Idea: Proof of seen other proposals
The first member that can show that its ticket is the highest with min N and max
M other tickets (all of which are verified to be valid). Gets to start the new
round, multiple may achieve this at the same time. But the network filters lower
ranking blocks and if two blocks survive each member will continue on the longest
chain. The higher the ticket, the easier this will be. The lower the ticket the
harder: but everyone is witholding its ticket? But if 2/3 is honest, under a certain
vrf threshold you are a loser, you

Lets consider a two role system. On each new tip (that is stronger) each member
rolls a vrf dice. If it is below a certain number they get to be the supplier
if it is above a certain number they can be gatherers. Suppliers release their
roll right away, dissemate throughout the network. The gatheres are racing to
combine a N-M amount of votes from the suppliers and dissemate that. When this
happens everyone starts another round.  

## Idea: Public keys that are harder to find then the vrf itself

## Idea: Voter + Ballot for every tip on the chain
Each member is allowed to  draw a ticket for every tip. An draw himself a
role for that tip. But new tips require a majority vote from voters. Voters
close down if a tip comes a long at a height that is higher then the voters height.  

vrf is always of n blocks down the common sub-graph of all tips n rounds in the past?
doesn't matter what tip? or the highest round with just one block (at least 2 rounds in the past)

## Idea: just vote counting, wait for enough proof  
Each block contains a draw and the draws of min N, max M draws of a tip at the prev
height. For each tip can gather enough proof and then draw an high enough ticket can
be proposed. The draw must be past a certain threshold, this threshold is adjusted
by looking at the average draw value of the last X rounds. If the prev tip is the
genesis block, no other proof is necessary: there is no selection/ranking?


Every member draws a ticket based on a deterministic merge of all block proposals
of the previous round. (ripple like), how to merge?


## Every proposer








## Blocks for 2 hours and then continues?

4762ad64: 12:11:02.657161 [TRAC] vote from '4762ad64' for block '80abf24d(27814)' proposed by '3a5a0c21' has caused a new tip: progress to next round
4762ad64: 12:11:02.657176 [TRAC] draw ticket with new tip '80abf24d' as round 27815
3a5a0c21: 12:11:02.657937 [INFO] --- drew proposer ticket! proposing block 'ddabdc72'
3a5a0c21: 12:11:02.658008 [INFO] --- drew voter ticket! setup voter for round 27815
3a5a0c21: 12:11:02.658018 [TRAC] blocktime is higher then zero, schedule vote casting in 100ms
3a5a0c21: 12:11:02.658192 [TRAC] block 'ddabdc72(27815)' proposed by '3a5a0c21': start handling
3a5a0c21: 12:11:02.658210 [TRAC] block 'ddabdc72(27815)' proposed by '3a5a0c21' was verified and of the correct round: relaying
4762ad64: 12:11:02.658480 [INFO] --- drew proposer ticket! proposing block '8d7bd3c9'
4762ad64: 12:11:02.658534 [INFO] --- drew voter ticket! setup voter for round 27815
4762ad64: 12:11:02.658550 [TRAC] blocktime is higher then zero, schedule vote casting in 100ms
4762ad64: 12:11:02.658710 [TRAC] block 'ddabdc72(27815)' proposed by '3a5a0c21': start handling
4762ad64: 12:11:02.658757 [TRAC] block 'ddabdc72(27815)' proposed by '3a5a0c21' was verified and of the correct round: relaying
3a5a0c21: 12:11:02.660229 [TRAC] block 'ddabdc72(27815)' proposed by '3a5a0c21' is new highest ranking block for next vote casting
3a5a0c21: 12:11:02.660324 [TRAC] block '8d7bd3c9(27815)' proposed by '4762ad64': start handling
3a5a0c21: 12:11:02.660345 [TRAC] block '8d7bd3c9(27815)' proposed by '4762ad64' was verified and of the correct round: relaying
4762ad64: 12:11:02.660657 [TRAC] block 'ddabdc72(27815)' proposed by '3a5a0c21' is new highest ranking block for next vote casting
4762ad64: 12:11:02.660760 [TRAC] block '8d7bd3c9(27815)' proposed by '4762ad64': start handling
4762ad64: 12:11:02.660777 [TRAC] block '8d7bd3c9(27815)' proposed by '4762ad64' was verified and of the correct round: relaying
3a5a0c21: 12:11:02.662240 [TRAC] block '8d7bd3c9(27815)' proposed by '4762ad64' is new highest ranking block for next vote casting
4762ad64: 12:11:02.662622 [TRAC] block '8d7bd3c9(27815)' proposed by '4762ad64' is new highest ranking block for next vote casting
4762ad64: 14:11:01.012962 [TRAC] blocktime has passed, and we are still voter, casted 1 votes
4762ad64: 14:11:01.013232 [TRAC] vote from '4762ad64' for block '8d7bd3c9(27815)' proposed by '4762ad64': start handling
4762ad64: 14:11:01.013542 [TRAC] verified vote from '4762ad64' for block '8d7bd3c9(27815)' proposed by '4762ad64': relaying
4762ad64: 14:11:01.013706 [TRAC] tallied vote from '4762ad64' for block '8d7bd3c9(27815)' proposed by '4762ad64', number of votes: 1
4762ad64: 14:11:01.013743 [TRAC] vote from '4762ad64' for block '8d7bd3c9(27815)' proposed by '4762ad64' doesn't cause enough votes (1<2): no progress
4762ad64: 14:11:01.013926 [TRAC] vote from '3a5a0c21' for block '8d7bd3c9(27815)' proposed by '4762ad64': start handling
4762ad64: 14:11:01.013947 [TRAC] verified vote from '3a5a0c21' for block '8d7bd3c9(27815)' proposed by '4762ad64': relaying
4762ad64: 14:11:01.014058 [TRAC] tallied vote from '3a5a0c21' for block '8d7bd3c9(27815)' proposed by '4762ad64', number of votes: 2
4762ad64: 14:11:01.014082 [TRAC] vote from '3a5a0c21' for block '8d7bd3c9(27815)' proposed by '4762ad64' caused enough votes (2>1), progress!
4762ad64: 14:11:01.027695 [TRAC] vote from '3a5a0c21' for block '8d7bd3c9(27815)' proposed by '4762ad64' while voter was active, casted remaining 1 votes before teardown
4762ad64: 14:11:01.027727 [TRAC] vote from '3a5a0c21' for block '8d7bd3c9(27815)' proposed by '4762ad64' caused enough votes, appending it's block to chain!
3a5a0c21: 14:11:01.013084 [TRAC] blocktime has passed, and we are still voter, casted 1 votes
3a5a0c21: 14:11:01.013338 [TRAC] vote from '4762ad64' for block '8d7bd3c9(27815)' proposed by '4762ad64': start handling
3a5a0c21: 14:11:01.031201 [TRAC] verified vote from '4762ad64' for block '8d7bd3c9(27815)' proposed by '4762ad64': relaying
3a5a0c21: 14:11:01.031365 [TRAC] tallied vote from '4762ad64' for block '8d7bd3c9(27815)' proposed by '4762ad64', number of votes: 1
3a5a0c21: 14:11:01.031387 [TRAC] vote from '4762ad64' for block '8d7bd3c9(27815)' proposed by '4762ad64' doesn't cause enough votes (1<2): no progress
3a5a0c21: 14:11:01.031517 [TRAC] vote from '3a5a0c21' for block '8d7bd3c9(27815)' proposed by '4762ad64': start handling
3a5a0c21: 14:11:01.031533 [TRAC] verified vote from '3a5a0c21' for block '8d7bd3c9(27815)' proposed by '4762ad64': relaying
3a5a0c21: 14:11:01.031607 [TRAC] tallied vote from '3a5a0c21' for block '8d7bd3c9(27815)' proposed by '4762ad64', number of votes: 2
3a5a0c21: 14:11:01.031622 [TRAC] vote from '3a5a0c21' for block '8d7bd3c9(27815)' proposed by '4762ad64' caused enough votes (2>1), progress!
3a5a0c21: 14:11:01.034103 [TRAC] vote from '3a5a0c21' for block '8d7bd3c9(27815)' proposed by '4762ad64' while voter was active, casted remaining 1 votes before teardown
3a5a0c21: 14:11:01.034125 [TRAC] vote from '3a5a0c21' for block '8d7bd3c9(27815)' proposed by '4762ad64' caused enough votes, appending it's block to chain!
4762ad64: 14:11:01.108096 [TRAC] vote from '3a5a0c21' for block '8d7bd3c9(27815)' proposed by '4762ad64' has caused a new tip: progress to next round
4762ad64: 14:11:01.108119 [TRAC] draw ticket with new tip '8d7bd3c9' as round 27816
3a5a0c21: 14:11:01.109512 [TRAC] vote from '3a5a0c21' for block '8d7bd3c9(27815)' proposed by '4762ad64' has caused a new tip: progress to next round
3a5a0c21: 14:11:01.109529 [TRAC] draw ticket with new tip '8d7bd3c9' as round 27816
4762ad64: 14:11:01.109547 [INFO] --- drew proposer ticket! proposing block '76eeae32'
4762ad64: 14:11:01.109616 [INFO] --- drew voter ticket! setup voter for round 27816
4762ad64: 14:11:01.109624 [TRAC] blocktime is higher then zero, schedule vote casting in 100ms
4762ad64: 14:11:01.109808 [TRAC] block '76eeae32(27816)' proposed by '4762ad64': start handling
4762ad64: 14:11:01.109874 [TRAC] block '76eeae32(27816)' proposed by '4762ad64' was verified and of the correct round: relaying
3a5a0c21: 14:11:01.111242 [INFO] --- drew proposer ticket! proposing block 'ffddb4aa'
3a5a0c21: 14:11:01.111296 [INFO] --- drew voter ticket! setup voter for round 27816



## Protocol
- Each member uses its clock to decide in which round we are, each round is a 5s
  time slot, e.g: unix time % 5.   
- Upon entering a round each member will pick the longest (highest priority) block
  in the previous (1, 1/2, 1/4 etc based on rank) round and draw a ticket using
  the VRF (scaled by stake).
- If a member draws a ticket below the proposer threshold it will broadcast a new block with
  the drawn priority and a proof of this. Multiple members may end up drawing a
  leader ticket for a slot, and multiple blocks are proposed that is OK.
- If a member draws a ticket below the notarizer threshold it will wait for blocks to come
  in and after some time, close the round by broadcasting a notarization. A round may
  lead to multiple notarized blocks, that is Ok. the mechanism is not for consensus
- If no-member draws a proposer or notarizer ticket (because part of the network is online)
  the round will produce no blocks and the system simply progresses to the next
- Over time the threshold is adjusted based on the number of blocks that have been  
  generated over the longest chain. Or the number of different proposers/notarizers there have
  been selected? adjusted by stake?

- Proposers wait for a certain amount of notarization before proposing a block
  so they can have some certainty. Proposing too early is risky and may cause
  you to lose your slot in the round. For how many notarizations do we wait? Should
  the proposer proof that it has seen N notarizations in the previous round?  
- Notarization keeps notarizing until it has seen an(other) notarization for
  this round was received.

- The relay policy heavily filters to prevent finality: selfish mining, nothing at stake:   
  Policy: each member is only allowed one block per time slot. Only relay blocks
  if no higher priority block for the slot has been seen. Blocks of older (-2)
  or newer slots (+2) are not passed over, to provide finality. Notarizations are
  relayed more liberal: one round in the past.

## Questions/Problems
- How do we prevent everyone to draw a ticket for each block in the previous round and
  just send the highest? Is this a problem? well connected proposers get an advantage with
  more options? Risk of getting behind? Lower prio block only count for 1/2, 1/4, of the
  weight, like in DFinity. If they don't send based on they highest they have a good chance
  to not win, since it is only 1/2 or 1/4 of the weight
