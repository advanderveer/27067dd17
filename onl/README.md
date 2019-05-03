# ONL

## Blocks, Flow and Finality
we can finalise a block when we know that majority is working on it as the tip or
indirectly by working on a tip that references the block. If a majority of distinct stake
"flows" through the chain, we know it is being worked on by the majority.

- every identity that wants to participate announces this
- by announcing this they put up some stake (>0) as a deposit (with a "join" tx)
- every identity cannot put up more stake then they own in their account balance
- they cannot spend the currency that the put up for deposit anywhere else in the system
- announcing participation costs an administration fee (to the treasury) that is
  easily earned back through honest participation.
- by announcing participation they commit themselves to using certain PK in the protocol
- by proposing blocks they bet their stake on the block's 'prev'
- this stake determines a block's ranking in a round, linearly
- this stake "flows" through all (indirect) predecessors of the 'prev' block
- if a block receive a flow of the majority of the stake each members observing
  it can finalise the block
- every proposer can submit the removal (leave-tx) of a member. If others agree that this
  is the case the block will be accepted and eventually finalized. Now, everyone
  can remove this identity from the participation list.
- the seed of each round's vrf is based on the last n-finalized block's token and the round
- members can gracefully leave the participation by sumitting the leave tx themselves. this
  will always be accepted.
- We always accept old blocks as a re-syncing mechanism
- Newer blocks are accepted out-of-order and resolve once the observing members enters
  the round. If the observing member is running behind it will work and just has the
  disadvantage of a large buffer of unhandled blocks that might fill up space. If the
  sending member is running ahead the block will most likely be rejected since we check
  if the referenced 'prev' is recent enough.

- (optimization) Can we adjust round-time by measuring (over time) the time it takes for the majority
  share to come in.
- (optimization) can the rounds be closed early if all members show up? Or even if
  the stake majority showed up? Yes, but if the majority didn't do this a fork might occur
- (optimization) can blocks be trimmed after some time? How do we know that other
  members don't need it anymore as proof of finalization? We can keep finalization
  references per block and throw away all other blocks.
- (optimization) we can trim any blocks that add majority stake for finalization
  beyond what is required. Specifically we can trim small stakes first to save
  space.
- (optimization) we can add a vrf threshold to limit the number of block proposals  
  per round. Since may increase the time it takes for blocks to be finalized but
  the top of the ranking should eventually cover majority stake holders

## Rounds and Timing
We group block proposals in time bound windows. Per round everyone gets to propose
one block. Blocks from old rounds are always accepted as a means of re-syncing out
of date minorty groups. New blocks are not accepted.

This requires reasonable synced clocks. For now we require NTP but we think it should
be possible to sync clocks precicely enough by looking at block times alone.

- Look into Marzullo's Algorithm: https://en.wikipedia.org/wiki/Marzullo%27s_algorithm
- And an extension on it that is used by NTP: https://en.wikipedia.org/wiki/Intersection_algorithm
- A golang implementation: https://www.slideshare.net/romain_jacotin/marzullos-agreement-algorithm-and-dtss-intersection-algorithm-in-golang
- Laports algorithm: https://lamport.azurewebsites.net/pubs/clocks.pdf

### Tip Based

#### Idea 2: Round times by chain history
We look at the timestamps in the chain history. But which tip?

#### Idea 4: Round ends when majority stake has casted a vote
Requires keeping of majority stake (intertwines layers) and which tip?

### Clock Based
Problem: What is the byzantine fault tolerance of it

#### Idea 1: Fixed round times
We look at the wall clock and decide in which round we're currently at. The
duration of the round is fixed. But how to adjust to network size/speed?

#### Idea 3: Some clock syncing algorithm
It uses messages from other members to sync a local clock. Probably Marzullo's
algorithm. How to adjust to network speed?


## Transactional Symantics
We would like to provide a kv abstraction that compares to the possibilities of
DynamoDB transactions (https://aws.amazon.com/blogs/aws/new-amazon-dynamodb-transactions/)
Or etcd v3 transactions: (https://coreos.com/etcd/docs/latest/learning/api.html#transaction)

It comes with conditions that can be specified as part of the transaction. This makes
it less powerful then Smart Contracts and Databases like FoundationDB but keeps it
simple and more of a building block. And not go full "smart contract" either.

- there are some special keys in an identities's namespace: 'balance'
- joins can be written as a condition on the systems key "balance" this key can
  never be deleted.

_How to prevent replay attacks?_ we can add some nonce, but it has to be on a field
or on the whole namespace but that strongly limits concurrency. Can we say that a
key is in write/read conflict if it mentioned between the chain
between "FinalizedPrev" and the "Prev".... hmmm sounds like we're gonna need
"A critique of snapshot isolation"
_Can we encode transactions as jsonnet merges?_ This is cool and flexible but the
vm has no way of limiting the runtime. As such miners might get abused with lines
such as: `std.foldl(function(x, y) x + y, std.range(1, 100000), 0)`. But the language
is discussed as becoming part of the Kubernetes config management which would
bring it into activ maintenance: https://github.com/google/go-jsonnet/issues/14
Maybe something for the future, but interesting to think of the internal structure
as being a json document. And I personally think that these people will quickly
escalate it into a full blown programming language, and at that point we might as
well run wasm.
_Can we encode changes as hcl2 merge?_ HCL2 is underway but does seem to be far
off: https://github.com/hashicorp/hcl2
_Can we use the 'life' wasm vm?_ Super flexible, maybe add as an addition to normal
condition trasnsactions? https://github.com/perlin-network/life

## Questions
- _what if members bet on the wrong prev?_ No-penalty because members cannot submit
  different vrfs per round anymore. They can if the reference different last finalized
  blocks.
- _what if happens in case of a fork?_ The minority will keep producing blocks but
  none of the blocks can be finalized until the majority shows up. When blocks arrive
  out of order and establish a new tip. This will resolve itself
- _What if block present different "last finalized"?_ If multiple are allowed we have
  nothing-at-stake problem as they can propose multiple blocks. Maybe have the weight
  be based on finalized block? If we discard we force the minority to catch up.
- _Do we accept blocks from far-future rounds?_ There is nothing in the sum weight calculation that takes
  round number into account. so a block submitted 100 rounds ahead doesn't suddenly get
  to be the tip. It does hover allow malicious users to grind rounds for high ranking
  vrfs and submit the highest one to gain a tip. But would only be actually received
  when the others enter that round as well, at that time it wouldn't be received because
  of the "has there been a block that could have acted as a prev earlier" that prevents
  such a block from being received.
- _Do we accept blocks from near-future rounds?_ Getting a block a round too early
  means that the member observing is still waiting for majority to come in. This means
  that the clock of the sender or the receiver is off. Maybe check if the timestamp is
  from the future also?
- _How to deal with many participants joining but not participating?_ this would cause
  the protocol to stop as no blocks would finalize which would preven these participants
  from being removed. Adding a penalty to not participating might be a good idea? Or
  eliminiation doesn't require finalization?
- _Can we determine this probabilistically through a sample of the network?_ Probably,
  but the 'majority' stake can already be a subset and one can check the percentage of
  acceptance for their own probability calculations
- _Can we adjust for propagation time?_ Or look at the timestamps of the highest ranking
  blocks and keep it as the average network time. If it always takes a long time
  for a certain member. Since we do no proof of work we don't need to adjust any
  difficulty but instead the limit is propagation speed of majority share blocks.
- _Add more incentive to check blocks before submitting?_ there is currently little
  punishment in publishing a block that isn't valid
- _How to deal with a large pool of leave ops?_ When a member times out, many other
  members may send in leave requests which are unnessary. Just add it as part of the
  block submission.
- _Can we snapshot the history?_ This page describes some solutions for bitcoin
  all the way at the bottom: https://eklitzke.org/an-overview-of-bitcoin-utxos.
   - Merkle Sets: https://diyhpl.us/wiki/transcripts/sf-bitcoin-meetup/2017-07-08-bram-cohen-merkle-sets/
   - Crypographic Accumulators: https://en.wikipedia.org/wiki/Accumulator_(cryptography)
   - Mimble Wimble: http://diyhpl.us/wiki/transcripts/scalingbitcoin/milan/mimblewimble/
- _Can we create kv abstraction where "_balance" is a special system key?_ Yes
- _Can there be multiple blocks in each round with a majority stake (be finalized?)_
- _what happens if stake is deposited or released in majority group_ nothing will change
  it will only affected if the block in it gets finalized
- _Can a identity commit to a pk VRF token without knowing what its gonna be?_ Then we can
  can base vrf solely on the round? Maybe make the vrf's randomness dependant on the token
  of the block that stores the pk commitment, this cannot be predicted. By committing the
  vrf pk the user gets assigned a (verifiable) random number for each token it generates itself.
- _What if we close a round only when majority of stored stake has proposed?_ This can cause
  the minority segment of the network to halt, but maybe that is ok?
- _Members can submit multiple blocks with the same token but  different writes?_ Yes and currently
  this grows the storage unbounded, but has not monetary incentive
- _Will we allow empty blocks?_ When we not allow this the protocol cannot make progress
  when there are no writes to include. Or it will at least include the coinbase.
- _Can identities create a whole lot of blocks in the future and then switch to
  another key?_ Make sure that they become invalid once the round is reached

## Future Plans
- Optional with WebRTC: https://github.com/pion/webrtc for NAT punching?
- Compile to WASM
- Allow update callers to specify a timeout to wait for updates to finish
- Allow views to take a handle that waits for a block with write to appear
- Allow this handle to have configurable certainty on a write

## Resources
- Ouroboros Praos, simple explanation: https://medium.com/unraveling-the-ouroboros/introduction-to-ouroboros-1c2324912193
- Creatking VRF with weaker primitives: https://eprint.iacr.org/2016/918.pdf

##
## Syncing Design
##

- _How are peers listed/found/selected/rotated?_ We will need them for syncing:
  Bitcoin has a custom peer discovery algorithm: https://en.bitcoin.it/wiki/Satoshi_Client_Node_Discovery
  Ethereum uses a Kademlia like DHT: https://github.com/ethereum/devp2p/blob/master/discv4.md
- _How are chains of peers diffed effeciently?_ If the chain is fully empty
- _How to catch up with live new blocks while syncing?_ must be fast enough, all
  stackup up in the Out-of-Order?
- _How does the validation of incoming blocks work?_ How to protect against wrong
  blocks being delivered?
- _Can a snapshot of the state be maintained and shared?_ to speed it all up?

- Make it a better situation then: https://github.com/ethereum/mist/issues/3738#issuecomment-390892738
  and https://github.com/ethereum/go-ethereum/issues/16251#issuecomment-371449572
- Bitcoin downloads from its peers: https://bitcoin.stackexchange.com/questions/55244/what-will-happen-if-a-block-is-lost-on-a-peer
- Bitcoin connects to 8 peers typically: https://bitcoin.stackexchange.com/questions/56775/how-many-peers-do-you-need-to-securely-synchronize-with-the-blockchain
- Ethereum Docs on Sync: https://github.com/ethereumproject/go-ethereum/wiki/Blockchain-Synchronisation

## Part 1 - P2P Layer
  This layer is responsible for providing the member with peer connections.
  - Bitcoin has a custom peer discovery algorithm: https://en.bitcoin.it/wiki/Satoshi_Client_Node_Discovery
  - Ethereum uses a Kademlia like DHT: https://github.com/ethereum/devp2p/blob/master/discv4.md
  - Then there is WebRTC (not covering the DHT part): https://www.html5rocks.com/en/tutorials/webrtc/infrastructure/
  - Perlin's Noise: https://github.com/perlin-network/noise
  - For datacenter deployments: Serf is a nice option

## Part 2 - Chain Diffing and transfer
 Given reliable connections to a number of peers, how to best sync over the required
 data and verify it validity. Can the same logic be used if the peer misses a few blocks.

 - _Snapshot syncing:_ Each block comes with a snapshot commitment, the commitments
   of (super majority) finalized blocks can be used as a starting points. Peers can
   be contacted to provide such a snapshot (_What incenstive to keep it?_) and this
   snapshot can be fed as the genesis state. (_What if a new snapshot is reached while old snapshot is being synced?_)
 - _Periodic block syncing_ Peers can nevertheless miss a block or catch up after
   a snapshot sync. So we will need to periodic block syncing by allowing the
   out-of-order structure ask peers for missing blocks or subchains.
 - _Old data trimming_ Data keys must actively be kept in the state or else they
   will be removed. This is aggressive but should provide a configurable upper bound
   to the stability

## Part 3 - Snapshots and fast Member bootstrapping
 Looking at the Ethereum situation it now takes days or weeks to setup a full node.
 As we're also building large chains we would like some snapshotting mechanism.

 Discussion for BitCoin here: https://www.reddit.com/r/btc/comments/5i269s/blockchain_snapshots_to_help_with_scaling/
 - TXO Commitment: https://petertodd.org/2016/delayed-txo-commitments
 - Merklix Tree: https://www.deadalnix.me/2016/09/29/using-merklix-tree-to-checkpoint-an-utxo-set/
 - Bitcoin Cash Snaphost: https://www.reddit.com/r/btc/comments/9pzyqn/bitcoin_cash_and_utxo_snapshots/
 - UTXO Commit: https://github.com/tomasvdw/bips/blob/master/BIP-UtxoCommitBucket.mediawiki
 - Flat ECMH Commit: https://lists.linuxfoundation.org/pipermail/bitcoin-ml/2018-March/000688.html
   blog post: https://www.yours.org/content/first-utxo-commitment-on-testnet-db7bf45bf83d
 - Ethereums StateRoot Commit: https://blog.ethereum.org/2015/06/26/state-tree-pruning/
 - Ethereums Merkle Patricia Trie used for this: https://github.com/ethereum/wiki/wiki/Patricia-Tree
 - IOTAL local snapshots: https://blog.iota.org/coming-up-local-snapshots-7018ff0ed5db


##
## Plan of Attack:
##

# 1. VRF Seed V2
- [x] The VRF seeds needs to a include a source of randomness that cannot be known
  when an identity commits to its VRF key. We will say that this will always
  be the Hash of the block that encodes the deposit stake
  of a member. It is verifiably random and not known until the block that encodes
  the VRF key of the joining is minted. The vrf token is now build of just
  Round+Pre-Commit Seed, so member can only rank exactly one block per round.
  _Rare Edge Case:_ In case of a fork the deposit can be in two blocks, at that
  point the identity may vote on both forks, this should be no problem as long as
  there is a rule that prevents stake deposits to be used too quickly (needs to
  settle in finalized block)

# 2. Write Nonce and Mempool V2
- [x] Each write is signed with a large random nonce that is generated by the proposer.
  It is in the proposers best interest to pick something that is definitely not
  being used before, probably by simply generating a large random nr
- [x] When minting a new block we will mask all writes in the mempool that are
  currently in the longest chain and already applied. This will prevent the
  replay of already applied writes (even if they have no reads).

# 3. OutOfOrder V2
- [x] Out-Of-Order will accepts a block for any round, blocks for future rounds
  will be stored and handle will be called when that round is reached. Creating
  blocks in a far future round has very little benefit through as its weight
  will be very low once the round is actually reached.  
- [x] Out-Of-Order is thread safe, multiple go-routines can call resolve and
  handle.
- [x] Handles are called in different go routines
- [x] All tests pass with the race detector

# 4. Finalization V1
- [x] Blocks in the chain keep track of the total deposit that has been stored such
  that it can be asserted if a (super)majority has voted their stake on it
- [x] If a stake holder has chosen a certain prev block to build on it indirectly
  votes on its ancestors. If a block has received a (super)majority of the stake in
  this way, it can be marked as finalized.

# 5. Out Of Order Sync V1
- [x] Both Mem and TCP broadcast implementations should allow writing back blocks to
  a peer when a sync message is received
- [ ] When testing the engine it should assert that when connection between two peers
  was broken for a few rounds, after reconnecting the should be able to sync up and
  reach consensus again.

- [ ] Engine minting should find the deposit block for its identity itself, not by
  hardcoding the genesis

# 6. Failure Mode Testing V1
- [ ] When the network splits exactly in half both sides shouldn't be able to
  finalize. When connectivitity between the two is re-estabilished, the missing
  blocks should be exchanged. Common tip calculated and should otherwise continue
  as expected.
- [ ] When a minority splits off it will continue on its own but not finalize, when
  it re-joins the majority segment there consensus shouldn't change while the minority
  adapts and takes over the majorities conensus
- [ ] When a member's clock skews them into starting rounds too early it should find himself
  working ahead in time which will cause blocks to arrive in rounds too early and have a
  low chance of being ranked highly (low sum weight)
- [ ] When a member's clock skews them into starting rounds too late it will propose blocks
  that arrive in old rounds that the majority cannot vote on making it a low chance
  of getting in the longest tip
- [ ] When a member's latency (tx/rx) is too low it will receive winning blocks too late
  and won't be able to deliver winning blocks in time for others to build on.

#Trimming V1
- [ ] The writes in finalized blocks can definitively removed from the mempool
  and any new writes that come in that are already in finalized blocks can be
  rejected right away.
- [ ] When blocks are finalized it should be possible to trim the blockchain of
  blocks in old rounds such that storage is bound to a single chain over time.

#VRF Threshold and expected chance to mint
- [ ] Each token must surpass a threshold that is calculated by looking at the
  average token difficulty of the last N blocks of the chain.
- [ ] Proposers are automatically removed if they fail to propose a block to the
  network, adjusted for the VRF threshold

- [ ] The algorithm can adjust or be configured to work in high-latency (WAN)
   and low latency (LAN) scenarios by changing the round time.
- [ ] New agents can join the network halfway during the execution of the protocol
- [ ] Rules prevent identities from writing to each other's keyspace, unless it
  transfers stake.  
