# 27067dd17
Decentralized leadership election using VRF and regular blockchain fallback

## Features
- Base case consensus at the speed of the network
- Worst case consensus using regular proof-of-work consensus
- Build in snapshotting and pruning of old block information

## Tradeoffs
- Sortition requires an up-to-date charter of miners, how can we bound the amount
without excluding new miners?

## Core Idea
At its core, this algorithm deploys a regular block chain with proof-of-work
Nakamoto consensus _but_ each block also comes with input for a VRF
(Verfiable Random Function) that allows 1-3 miner(s) to win a lottery ticket.
This ticket lowers the required difficulty for them such that the block can be
mined almost instantly. Miners that did not win the lottery instead get,
from the VRF, an advisory wait time before starting their mining effort.
This prevents the network from wasting resources as a lottery ticket will always
outpace regular mining effort.

### what lottery?
The lottery is a draw from a charter of miners that is updated on every block.
The charter contains a numeric value for each miner that is participated in the
protocol. The value is based on the miner's value to the network (stake/utility/speed)
and miners with a higher value are more likely to win the lottery (with a max value).
New miners can freely join and leave the charter through a broadcast. Miners that
do not make use of winning tickets are removed from the charter.

### What if no leader lottery ticket is drawn?
The algorand sortition doesn't guarantee that a leader is chosen: "Appendix B of
the technical report [27] proves that choosing τproposer = 26 ensures that a
reasonable number of proposers (at least one, and no more than 70, as a plausible upper bound)
are chosen with very high probability (e.g., 1−10−11)."

### What if many leader lottery tickets are drawn?
As in algorand, we apply filtering on the gossip layer. Members do not gossip
blocks if they've seen one with a higher ranking priority

### Charter management?
- Can users be banned?
- How are is a members value exactly calculated?

### advisory wait time?
The lottery allows for extremely fast confirmation time if the miner that has drawn
the top ticket is currently online. If this is not the case the network should fall
back to regular proof-of-work consensus. As such, each member is advised to wait
for the lottery winners to mine a block before attempting to create one themselves.
This saves regular miners a lot of work because the lottery ticket lowers the
required difficulty to such amount that normal mining will never outpace
the lottery winners. This waiting time  may later be enforced using an
VDF (Verifiable Delay Function) when the research community
(https://vdfresearch.org/) has found a secure implementation.  

The proof-of-work is callibrated such that it should be more profitable to wait
a certain amount at first. But after 30 sec it should become profitable to start
mining

### speed and storage?
If a miner wins the block lottery it create a new block immediately. As such if
every miner that is picked from the charter stays online the chain advances at the
speed of the network. During low traffic times this might cause many small transactions
and cause the storage overhead to grow quickly.

We add a build in pruning snapshot mechanism, every N blocks a new snapshot
watermark is reached. Every miner must then encode its hash of the snapshot at this
height until a new watermark is reached. Downloaders of a snapshot should verify that
a super-majority of blocks after the tip of the snapshot have the hash of it encoded.

### longest chain consensus?
As with a regular block chain, members continue on the tip with the most cumulative
solved difficulty (weight) behind it. Ticket winners have encode the same solved
difficulty for their block but include their VRF proof to show for it.

## Resources
- DFinity, VRF + Beacon: https://blockonomi.com/dfinity-threshold-relay-consensus/#Random_Beacon_Layer
- Algorand, VRF for sortition: https://www.algorand.com/sites/default/files/2018-11/Byzantine.pdf
- June Network, Verifiable Random Time: https://jura.network/Whitepaper%20of%20JURA.pdf
- Witnet, VRFs: https://medium.com/witnet/cryptographic-sortition-in-blockchains-the-importance-of-vrfs-ad5c20a4e018
- Advanced VRF/zk-SNARKS: https://eprint.iacr.org/2018/1105.pdf
- DFinity Implementation in Go: https://medium.com/@wanghelin/why-dfinity-is-my-favorite-consensus-protocol-for-implementing-dex-c315dc0712ee

## TODO

- [x] Show that a VRF can draw a priority based on stake
- [x] Show that this can be verified by any actor
- [x] Show that the priority can determine the required PoW
- [ ] Show that this leads to a longest chain with acceptable forking
- [ ] Show that with filtering this can be reduced heavily

- Show that VRF can partition a list of members into block difficulties
- Show that a members claimed ticket can be verified by everyone
- Show that this position can segment the charter in required difficulty
- Show that this difficulty can be verified by everyone

## Timeslot idea
Consider: Reasonably synced clocked across the network (NTP, +/-50ms). Or an
implementation of something like:
 - https://arxiv.org/pdf/1707.08216.pdf
 - https://www.distributed-systems.net/my-data/papers/2006.selfman.pdf
Or a subset of the network is picked to perform a PoW to act as as clock? Or everyone
adjusts based on network time of seen blocks. Or a setup like:
https://github.com/chainpoint/whitepaper/issues/6

- Each member uses its clock to decide in which round we are, each round is a 5s
  time slot, e.g: unix time % 5.   
- Upon entering a round each member will pick the longest (highest priority) block
  in the previous round and draw a ticket using the VRF scaled by stake.
- If a member draws a ticket below the threshold it will broadcast a new block with
  the drawn priority and a proof of this. Multiple members may end up drawing a
  leader ticket for a slot, that is OK.
- If no-member draws a ticket (because part of the network is online) the round will
  produce no blocks and the system progresses to the next
- Over time the threshold is adjusted based on the number of blocks that have been  
  generated over the longest chain. Or the number of different leaders there have
  been selected? adjusted by stake?
- The relay policy heavily filters to prevent finality: selfish mining, nothing at stake:   
  Policy: each member is only allowed one block per time slot. Only relay blocks
  if no higher priority block for the slot has been seen. Blocks of older (-2)
  or newer slots (+2) are not passed over, to provide finality.




## Slot based VRF, Ouroboros Gensis: (https://eprint.iacr.org/2018/378.pdf)
On time in Ouroboros: https://forum.cardano.org/t/proof-of-stake-timeslots/11300/16


- slot-leader election: Each party Up checks whether or not it is a slot leader, by locally evaluating a verifiable random function (VRF, [15], modeled by FVRF) using the secret key associated with its stake, and providing as inputs to
the VRF both the slot index sl and the so-called epoch randomness η.
If the VRF output y is below a certain threshold Tp—which depends on Up’s stake—then Up is an eligible slot leader
- A delicate point of the above staking procedure is that there will inevitably be some slots with zero
or several slot leaders. This means that the parties might receive valid chains from several certified slot
leaders. To determine which of these chains to adopt as the new state of the blockchain, each party collects
all valid broadcast chains and applies a chain selection rule "maxvalid-"bg. In fact, the power of the protocol
Ouroboros-Genesis and its superiority over all existing PoS-based blockchains stems from this new chainselection rule which we discuss in detail below.
- Thus the new rule substitutes a “global” longest chain rule with a “local” longest chain rule that prefers
chains that demonstrate more participation after forking from the currently held chain Cmax. As proven in
Section 4, this additional condition allows an honest party that joins the network at an arbitrary point
in time to bootstrap based only on the genesis block (obtained from FINIT) and the chains it observes by
listening to the network for a sufficiently long period of time.
