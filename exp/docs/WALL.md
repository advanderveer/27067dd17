# Vote Mining

- On every tip, all members can VOTE infinitely many times with a VRF and a seed
  based on the tip and a number of their choosing. It is allowed to mine a number
  that results in a very hight token but there is no incenstive to do so. It is
  ok if only honest nodes perform this action.  
- VOTE messages are small and fixed sized and can be broadcasted over UDP. Losing
  a few is not a problem, as long as the majority observes the votes.  
- Each member keeps a filter that will ignore votes from members that sent them
  in too quickly (the wall). It basically acts as a DDOS protection to make sure
  members do not flood the network with VOTEs.
- Honest nodes will only vote on the strongest tip. Non-honest votes might pick
  older tips to vote on but cannot get other nodes to accept more then one vote
  from them per time period.
- In large network the wall can be bounded in size by only accepting votes that
  have drawn a reasonably large VRF token.

- Each member can use each tip VOTE as a nonce to hash another block. If it passes
  the difficulty prescribed by the tip the block can be broadcasted.
- Everyone must accept the block if it has also seen the vote that was used as a
  nonce. If the new blocks total strength is bigger then the old tip, voting on
  the old tip will stop.
- In large networks the number of miners can be constrained using the vrf that is
  drawn from the tip.
- In general, the miners are waiting on new votes to come in to try as nonces, the
  hashing time is negligible. Instead there is incentive to collect as many votes
  as quickly as possible: being well connected.
- Tip strength is based on the solved difficulty and the vrf. In that way the network
  should converge faster if two blocks are produced at the same time.

- Relay only votes that are votes for our strongest tip. This should further amplify the network
  convergence towards a single tip. Non-honest nodes can relay if they want so it might
  reach most of the network but thats ok. As long as as small number of votes arrives
  non-honest miners will have a small chance of mining a block and honest nodes will not
  accept it anyway.

- Whenever new there are more miners or voters the process should be happening quicker
  so adjusting the difficulty on timing between blocks seems feasible. As long as we
  assert block times to be near the network time mean
- Can we keep our local clock in sync with network time by asserting vote messages times.

## Questions/Problems

- _How to keep nodes that didn't accept blocks while majority did up-to-date_. We
  keep unaccepted blocks and if another block builds on it we accept them both. This
  means we need to keep votes for all blocks, even the ones we don't know about?
  Can we switch over to another tip if we don't receive votes for the tip we're on
  after a certain time? Side-channel sync every n amount of time, how to select the member?
- _How does the voting generation ensure finality?_  Voting on old tips is not possible as no-one
  will relay those messages. Mining old tips is not possible as no-one will have the
  votes it is build on.
- _How to prevent private vote mining?_ One could generate infinite votes privately
  and only send them out when a block could be mined. This means we need a bound on
  the vote nr that is send around: e.g one can generate votes but not send them out
  but not faster then the network. (does this mean we're back to time synced rounds?)

## Algorithm 1: Tip switching
Whenever a block is accepted that results in a new strongest tip we will start
voting for that tip: relaying any other vote for that tip and generating votes
ourselves.

  Worst Case: All votes for any tip always manage to reach all members

## Algorithm 2: Vote mining
Whenever a vote comes in [for ...] we attempt to mine a new block for our current
tip by using the vote as a nonce. Non-honest nodes may try to mine for multiple
tips but a block message will also only be accepted by everyone once ever time
unit so sending it is a risk.

 Worst Case: Everyone always mines all tips

## Algorithm 3: The Wall
Each messages comes with a vrf and tip. Messages are only accepted if the vrf
is above the threshold prescribed by the tip. This provides a bound on how many
members in the network can participate for a certain message type. Whenever a
message comes in the wall will start a timer, if another message from the same
member comes in the message is discarded and the timer doubled. Members who are
flooding the network will therefore be blocked.

  Worst Case: Every vote/block for distinct tips is unfiltered

## Algorithm 4: Difficulty & Time keeping
Each block encodes the time it was mined. The block is only accepted if the time
doesn't diverge too far from the mean network time. The time at a tip is the mean
of times of all votes (that also encode a time). Based on the mean time between
blocks the mining difficulty is adjusted and VRF thresholds enabled.

  Worst Case: Very sudden member increase


## Proof 1: Worst case should be correct and secure but slow
- 50% of the members vote and mine all tips (non-honest)
- 50% of members just vote and mine one tip (honest)
- (out of order accepting of blocks?)
- Every message is always relayed to everyone
- No VRF threshold or DDOS wall
- Difficulty is adjusted as normal

## Proof 2:
