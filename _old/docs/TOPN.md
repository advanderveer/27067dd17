# VRF Top as weight, no rounds

Every block represents a 'vote' aspects and a 'data' aspect. we accept the vote
tentatively and keep the top-N votes for a prev value. we only relay messages for
that prev value if it ranks in the top N. i.e for relaying we do not need to know
if the previous block existed or not. we keep rankings for phantom blocks

When the block itself comes along we resolve these votes and and any other blocks
that were waiting for it.


# VRF Top as weight, tip selection on heaviest chain

Each vrf draw should strengthen a certain chain by the rank of the draw. As more members
pick a tip the chance of high ticket drawn for that tip increases and enables
the consensus. i.e Statisically the chance is very small that tips the minority
picked will have tokens that reference them in the top-N. Tips that the majority
picked will have a much higher chance of ending up in the Top-N

This means that the current heaviest tip can vary per member but after N rounds
there should be certainty on blocks propose a few rounds back.

Members always relay blocks if they rank in the top-N. if the prev doesn't exist
the will simply not hand out points to upgrade their heaviest tip. Gradually the top-N
will now be tips that the member doesn't know of at so it can no longer propose
sensible blocks and effectively leaves protocol. It is up to the member to spend
effort in getting back up-to-speed.

Rounds are time windows that stay open as long as stronger blocks arrive. if the TTL
expires the round is closed and after a fixed time members will start proposing blocks
for the next round. Until a round is closed no proposals for the next round will be
accepted. Blocks that are proposed too-early or too late are discarded.

(bonus) sync clocks based on the top-n blocks round times that are observed and
(bonus) top draw identies might decide to close the round early
(bonus) light nodes vs full nodes: light nodes just participate in rounds (keep a top-N and relay) but do
not keep a chain and distribute weight to determine new tips.


@PROBLEM how to prevent people from mining new identies just to win the round (sybill attack?)




# Top-N Block weight
- Everyone can proposes a block with VRF: Prev and sees is from the current tip
- A block is proposes for a certain round and everyone just keeps the Top-N proposals of a round
- Everyone just relays top-N proposals for that round
- Every time a top-N comes along the ttl for the round is reset
- After the ttl is expired, accept no more tickets and close the round
- after the round is closed, wait a fixed amount of time, then propose a new block. The blocks is proposed
  on the current absolute heaviest tip we know of.
- VRF token ranks a proposal in the round and gives it individual weight.
- The

# Round-layered
- Everyone can propose vrf ticket: always have round input
- Everyone keeps the top-N tickets: constant size
- Only relay top-N tickets: filter most messages
- tickets are small and fixed-sized: send over udp
- every time a top-N ticket comes along reset the ttl timer: propagation dependant time window
- after the ttl has expired, accept no more tickets and close the round: finality on the ticket set at each round
- only accept new round tickets if the old one is closed: rounds progress as the speed of the majority

@PROBLEM vrf seed?
- when the round is closed, wait a fixed amount of time, then propose a new ticket with the seed as the
  previously top ranking ticket in the previous round?
- only accept a ticket if the the prev ticket was found in the last round?
- What if it arrives at a place where the prev doesn't exist?
- We could sample the top-N somehow, but this would enable mining to a certain extend.
  but only soo many options? But if there is no incentive and the network is always outperforming

At this point the (honest) majority of the node should have a new bucket of tickets
every N seconds. This bucket is reasonably consistent between members but unlikely
to be the same 100%.   

## Block creation  

## Bonus
- (bonus) sync local clock tick on ticket messages

# Top-1 VRF
- Everyone can propose a block with a VRF: always something to propose
- Everyone only keeps the highest ranking block: constant size memory
- Only relay if the blocks rank higher then is currently known: filter most messages
- Every time a new highest ranking block comes along, reset the ttl: propagation dependant time window
- Accept only new round blocks if the current ttl has expired: no-one can move faster then the majority
- If the ttl expires, ad the highest ranking block to our chain as the new tip

# Top-N VRF

- Everyone can propose during a round: always something to work with
- Everyone only keeps the top-N ranked by vrf token: bound on memory  
- Only top-N is relayed: this will filter most messages

Tip selection
-

Round Closing  
- Every time a new top-N comes in the ttl of the round is reset: propagation dependant time window
- Accept only new blocks for the next round if the ttl has expired: force every-one to wait a reasonable amount
- After the ttl has expired, wait a fixed amount, then send a block proposal for the new round. honest members
  will set the prev to the heaviest tip they know.


- (bonus) adjust ttl timer on the nr of members
- (bonus) We sync our pulse clock on the top-N: gain reasonably synced round pulses: why?
- (bonus) Add a proof that showes that the proposer's top-N: is reasonably similar to ours: why?

- With consistent broadcast this should ensure the Top-N is "reasonably" consitent after some time

ideas:
- find a way to fingerprint the top-N such that it proves that a member waited
  long enough
- find a way to sync clocks
