# TopN VRF

- Every member can submit a block with a VRF rolled token marked for a round
- the block has one prev and the VRF seed is based on the prev block's token (crypto random)
- each honest member will pick the current heaviest chain as its prev for the block
- each member collects the top-N ranking blocks for the round
- each member only relays if a blocks ranks in the top-N
- when the round is closed, based on the ranking in the top-N weight points are handed out to the prev block(s)
- if the prev block doesn't exist the top-N is still kept, only for relay purpose, weight can be added out of order

## Time, Finality and Selfish Mining
At its core, timing of the round prescribes the forking rate. If its 0 everyone
can just instantly continue building on its current (very incomplete) view of the
top-N. If its too long we're waiting unnecessarily for vote (points) that will
never arrive, slowing the network down.

Bitcoins 10 minute is a long time, Litecoin is callibrated at 2.5min. Ethereum
is more aiming at 10 seconds but rewards orphan blocks.
@see https://medium.facilelogin.com/the-mystery-behind-block-time-63351e35603a
on how block time chosen in general
@see https://ethereum.stackexchange.com/questions/5924/how-do-ethereum-mining-nodes-maintain-a-time-consistent-with-the-network on how ethereum validates time

A paper in 2013 (https://www.tik.ee.ethz.ch/file/49318d3f56c1d525aabf7fda78b23fc0/P2P2013_041.pdf)
shows mean block distribution in 12.5 seconds.

Bitcoin on 40min propagation delay: https://www.reddit.com/r/btc/comments/9ynhtf/todays_64_mb_sv_block_took_40_minutes_to/

In our blockchain the incenstive of being included can be lower as the required work
is very low. As such the penalty for making an orphan block is less severe.  

### Idea 1: Next round time by looking at the chain and block arrival diffs
What if we set the next round time to the mean difference of block creation and
arrival times in the top-N? Or adjusted by new tip of closed round? The spread of
points in the chain graph illustrates how the network is doing on propagation. if
points are spread out this means members got way different views on the network.

Insight 4: The top-N can be analyed to see how high the vrf tokens are to estimate
how many members tried to propose a block. This can be used to adjust vrf thresholds.

Insight 3: Its in the members best interest to close (and open) rounds in tandem with
the rest of the network. Else it will be out-of-sync consistently and not be able to
propose any useful blocks.

Insight 2: Low round times will cause orphan blocks with points, we can detect this
by analysing the chain and adjust the round time.

Insight 1: Top-N is basically a sample of the network (maybe skewed towards members
with more stake but we trust them more anyway). So we can use that to sample
time characteristics of the topology also.

- _What if members 'bet' on my blocks in a round_ Not allowed, only first pick is accepted
- _What happens with timing if members 'bet' on a block too early_ Normalize by the correctness
if its bet? timestamp should still be reasonable (after the closing time of our round)
- _Can the network be attacked by many out-of-sync nodes_ ?
- _Can the network become biased towards low latency nodes_ ? effectively centralizing the algorithm?

## DDos& Nothing at stake
How do we prevent against people making blocks for all tips its knows
- every round keeps a bitmap of all pk's, bound by a vrf threshold
- each identity is only allowed to propose one block per round

## Economics, Sybil and key grinding
Attack 2: Create many identities in general to increase the chance for a reward.
_Solution 1_: Participating  in the mining process requires putting in stake. This
stake can only be acquired from a pre existing amount of money in the system
(minted in the genesis block)

Attack 1: Create a new vrf identity at every round and checking if the resulting token
is better.
_Solution 1_: The public key must be off a certain difficulty and cost significant resources
to create.
_Solution 2_: With proof of stake this problem can be solved as long as it takes multiple rounds
to make a stake deposit usable and economically certain. It should also take significant time
to return a deposit.

Economics:
- 0 stake, means 0 chance: making many 0 stake identities is useless (1000*0 is still 0)
- stake distributed across many block proposers must give the group the same amount
  of chance as having the stake on one identity. This removes the incentive to
  make many identities (10*100 == 1*1000).
- stake can be transferred from an existing member
- there is a central treasury identity that funds development and can hand out initial stake
- proposing a block that is selected will reward stake to the user and the treasury
- Stake can only be unlocked after significant time. Stake is only usable after significant time
- (optional) punish miners if they misbehave by slicing there stake

@TODO can stake be locked into block proposals themselves such that it is not economical to
make multiple by spreading the stake? i.e chance of winning a round is the same whether the
stake has been spread over multiple tips or all in one tip. If the non-winning tip/prev is staked
there is a significant risk of losing it.

## Missing block re-syncing
if a block arrives from a peer while the prev isn't known, we cannot assume that
it has this block. Members will always relay if a block is in the top-N of that
round. So how are we gonna get the missing block?

- Maybe add a commit step or message? Why not use that to transfer the block?
- Separate voting from block messages, block messages separate from rounds?
- Send headers first, block messages can complete missing headers.

@TODO

## Domains and multi-level system
Imagine how many devices there are in the world. If we truly want to connect them we need
to think in domains. A worldwide domain might move slower, while lower level domain can move
much faster. Maybe with their own rules? This allows for isolation of faults

@TODO

## Bounded space for chain storage?
If we want the chain to existing for 100s of years there needs to be a bound on storage
of the data.

- Idea: measure of activity? force people to move out or in? clean up old data?
- Idea: sharding? proof of storage?

@TODO

## Do not send full blocks if there referenced transactios are already in the mempool:
https://www.youtube.com/watch?v=GYEZ52WVKEI&vl=en

## Other ideas
_What about no incenstive_ participating may be so cheap that no incentive is
necessary at all?

_Broadcast with DHT?_ it may be important for use that messages dissemate randomly
throughout the network. can a DHT help in that regard?

_Isn't it wastefull most of the block data will never make it into the top-n_ can
we send headers via udp first or something? Can we have blocks as optional attachments
to votes that travel over udp: no then we have votes without blocks polluting the ranking

### Other remarks
- we would like to incentivize synced clocks, there are many ways to do this the age
of GPS and NTP.

- synced clocks might be accomplished using the ranked blocks alone. with existing
protocol.
- we would like to use the round finality for optimistic consensus
(DFINITY style) not true consensus.
- base points on the difference between block creation time and arrival time? this
incentivizes fast propagation speeds (but do not allow empty blocks?)
- we need block times for difficulty adjustments anyway. what else can we do with
the block timestamp.
- blocktime must be higher then the prev block's timestamp (Etherereum rule)



## Algorithms

Members always accept blocks for a round if it isn't closed yet and no other
block from that member have been seen in that round. ~~Specifically, the prev
doesn't need to be present for the block to be accepted into the topN~~. _Can malicious
users poision the topN with blocks that point to prevs that don't exist?_ No, this
would require a majority stake in the network as the VRF sampling makes this unlikely
_How to verify vrf if the prev isn't present?_ Not possible, the prev block
must exist for the the new block to exist

In the chain structures votes are assigned and the tip is advanced. For the chain
to accept new blocks and its tip to advance the prev DOES need to exist. Each append
comes with a weight that is based on the ranking at which that member observed the
block in it's round.

This makes the vote distribution not consistent between chains _How does the eventual
consistency of this work?

_What if member keeps stuck on old tips?_

_What happens if a block arrives and changes the rank of other
blocks in that round_ When the next round is closed we need to have all blocks
synced? No, we need something that is optimistic and eventual consistent (whp)


## Questions
Why can't we time box per block? Each member is only open to blocks that build on
a prev for a certain amount of time.

Each member only  A tip only stays open for a certain amount


## Stake, Economics, Sybil and Nothing at Stake
Stake is locked in the prev block; members bet on a tip. If the tip becomes the
longest chain. The stake is released, sometimes this causes losses but this should
smooth out over time with mining rewards. It decentivizes building on old tips,
on multiple tips and building on tips too quickly. (nothing-at-stake)

_But how to this bet, engineering wise?_ Keeping all the blocks in the chain
is a storage nightmare. Specifically betting on many tips, how to record if the
block itself (or even the tip) is never committed as part of the longest chain?
- Stake as a transaction to the treasury that can be referred to proof stake. Must
be in the longest chain.
- this can later be reclaimed by another transaction if your bet was correct. If bet wasn't
correct you can never reclaim it. But this requires upfront knowledge of what you're gonna
bet on?
- any honest node can report miscondect and reap a reward? Only possible with round
setup as we can proof that a user signed two blocks with the same round nr. But with
rounds it is fully feasible to make this impossible anyway.
- whenever a node receives a block but the prev is not the heaviest tip (in their opinion),
honest nodes will submit a burn transaction.  

The chance of a block being in the topN (and becoming the longest chain) depends on
how much stake pas placed. With 0 stake, the chance is 0. And spreading
stake over multiple blocks doesn't increase the overall chance of the identity
receiving a reward. This decentivizes the creation of many identities or many
blocks to increase the odds (sybil)

High stake members may have a high chance of being the topN but it shouldn't be
a 100%, must be balanced based on the overall economics. Rich people can still get
richer but with diminishing return such that getting a majority share
is very hard (impossible?).

If members bets too early they are likely to pick the wrong prev and lose their
stake. If members bet too late their block is unlikely to be in the top-N of any
other member and never be picked for the longest chain.

_How do we keep track of this with millions of miners?_ Can we lock and release stake
just for the topN?
_Stake needs to be verifiable during voting?_ If members bet on multiple
tips we need to be able to substract their balance even if the tip doesn't become the
longest chain.



##

- Every identity can only propose 1 block per round
- A round is a time window each members enters at the same time
- Per round blocks are ranked using the blocks' token
- Each members draws their block's token using a VRF
- The seed for the VRF is based on the prev block's token
- blocks are only accepted if the prev block exists and token is valid
- each member keeps a top-N of candidate blocks per round ranked by token draw
- blocks are only relayed if the token ranks higher than other candidates in a round
- the tip is the heaviest chain of blocks with weight based on the rank per round
- as new blocks move into old rounds the now heaviest tip may change
- if a prev block doesn't exist the proposing peer must provide it
- the chance of drawing high ranking vrf is based on stake, 0 stake == 0 chance
- when block becomes part of the longest chain a currency reward is given to the proposer



Edge case:
- 50/50 network split
- minority has highest ranking block
- majority has highest ranking block

- What should happen if a minority tip leads to a very high draw in the new round.
  - almost no-one should have the minority tip so the chance of an high ranking block based
    on it is very low
- What should happen if the network splits in two and each side goes on with there own
  tips for some rounds. How do members sync up their chains?
- What if some member just happened to miss the highest ranking block in a round?
  next round it will propose a block


- if the prev block doesn't exist but the referencing block allegedly
  ranks higher the receiver should ask the sender to proof this by sending
  the prev block over that it was based on. Some members may try to announce a fake block to the
  network with a mined 'prev' value. If the prev block indeed exists and ranks
  higher we add it to our chain and relay the new highest ranking block. This may
  cause us to switch tips, but our proposal was already send. This indirect syncing
  of blocks can happen after the round has closed.
  - if an invalid prev proof is provided the new block is discarded
  - if a valid prev proof is provided the block is added to the chain so we do
    not need to sync it again if others vote on it
  - if the prev proof is not our tip, its is not relayed through

- blocks are only relayed if it ranks higher then existing candidate blocks
- the round closes ... @TODO
- at round-close the highest ranking block is appended to the members chain
- members will now propose a block with their (new) tip as prev

- At this point we either are on the tip of the majority, or on a minority tip.
  by syncing new round 'prev' blocks that rank higher the minority tips. If the
  the minority has the highest ranking tip (this is possible) the new round may
  take forever.

- some peers will now propose a block with a prev thats not their tip.
- some peers will propose a block based on a prev that some other peers don't have in their chain
- some peers will propose a block on a prev that no-one has ever seen
- How do we get the wrong minority to switch to the majority, it should assume that
  the top ranking vote's prev has some merit. Else it wouldn't end up there so
  it needs to source it from somewhere.  

- The honest majority should have a pretty complete topN




- if a prev block doesn't exist, the peer that proposed the referencing block
  can be asked to provide the chain leading up to it. and continue working on it
