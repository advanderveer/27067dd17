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
- _Can we create kv abstraction where "_balance" is a special system key?_
- _Can there be multiple blocks in each round with a majority stake (be finalized?)_
- _what happens if stake is deposited or released in majority group_ nothing will change
  it will only affected if the block in it gets finalized
- _Can a identity commit to a pk VRF token without knowing what its gonna be?_ Then we can
  can base vrf solely on the round?

## Resources
- Ouroboros Praos, simple explanation: https://medium.com/unraveling-the-ouroboros/introduction-to-ouroboros-1c2324912193
- Committing to something upfront: https://eprint.iacr.org/2016/918.pdf
