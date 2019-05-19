
### V2 Design
- There is a special staking address.
- Before proposing blocks a transfer has to be made to this address.
- This transfer is referenced in a block proposal as stake.
- The deposit transfer can only be used for a certain amount of time.
- After that time a new deposit transaction needs to be made.
- After even more time the first deposit can be reclaimed.
- It uses the Ouroboras Genesis threshold function
- All usable recent deposits are summed to determine the total stake.
- Blocks are proposed for a certain round.
- Only the first block from each identity is accepted in a round.
- We take the DFINITY longest chain rule
- Each block come with the blocks that lived in the round next to the prev
- The stake and time for these blocks are also encoded in the longest chain
- Stake from the new block and its witnesses flow through the ancestory
- If all stake at an ancestory has been accounted for it can be finalized
- A finalized block acts as a checkpoint that new members can bootstrap from
- The timestamps in witnesses and blocks can provide a dynamic round time

### R&D Questions
- RQ1: How do deposit transactions look like, how to securely time limit them?
    - Idea 1: A transaction to of which the output can only be spend after a certain
      amount of blocks. This height difference must be larger then the time that
      deposit transactions can be used for minting blocks. _But can we used this
      to slice the user if proof of malpractice is provided?_ in that case it will
      be (partially) spendable by some-one else before it unlocks
- RQ2: Does walking back for total stake work in practice?
- RQ3: How quickly can be expect finalization, if at all?
- RQ4: How does the round-time adjustment take place? Can it be attacked?

### Engineering Questions
- [x] Just one PK, simplicity is king -> Signature is VRF, is ID
- Block syncing with peers needs to be more efficiency

### Attacks
- Nothing at stake/Double voting - Not possible in this system.
- Long range - Witness weight and finalization
- VRF key grinding - VRF randomness is purely based on the prev block, if
  a pk would be grinded it couldn't be used because it requires at least one
  block to deposit the stake that is necessary.
- Stake grinding - VRF token has no free parameters, it is purely based on
  the prev block's token

### TODO

- [x] try a VRF based signature on the transaction.
- [x] create a signature that creates and validates a deposit transaction
- [ ] create a simulation that tests finalisation and total stake calculations
- [ ] write down a round time adjustment protocol


####
### Old Brainstorm Below
####  

PoW has three properties
- It takes time:    Every block height stays open for certain amount of time  
- It takes capital: One must proof that it can unlock funds to bet on a block
- It is risky:      Other members can proof that the stake was used maliciously   

With the collects we can encode info about:
- the time it took for each better to send in a block: hopefully adjust the
  round time.
- the nr of bets that were high enough to pass the threshold and adjust the
  threshold over time.

# V1 Problems:
There are a few problems with the first implementation:

- _Multiple PK's to manage_ One for block signing and one for VRFs, the latter
  of which needs to be registered first.
- _Participants need to register/deregister:_ To commit to a PK and to to get
  randomness. Needs a way to auto-register if user is not active.
- _Fixed, unflexible round time_ The round time is now static, bound to a clock
  that needs to be synced up.
- _Multiple blocks can be send in with the same ticket_ It takes no effort to
  send in another block with the same token.
- _Consensus layer mixed up with KV layer_ We use the kv layer to store stake
  values. This makes it hard to to secure properly.
- _Block syncing logic is inefficient_ We ask all peers and they all send a copy


# Rules

- Every members have a roughly synchronised clock. For now this is used synced
  using NTP but in the future there might some algorithm that can adapt to the
  nr of members or the state of the network.
- By looking at the clock the honest majority enters each round. We split up
  time into discrete time windows in which the majority will accept and build
  on blocks.
- Everyone else can submit blocks in the past or present at will. But the majority
  will only accept and build on the heaviest tip from the current round down.
- The honest majority will not accept newer blocks until they enter the respective
  round. non-honest members can submit for new rounds but once the majority enters
  the round after it, it will be unlikely to be the heaviest chain.
- Blocks from old rounds are always appended and can change the ordering of blocks
  in that round and the rounds after it.
- Blocks for newer rounds are stored out-of-order and accepted once members enter
  that round.
- Blocks that depend on non-existing prev are also stored out-of-order and are
  received when the prev block has been received.
- Each block comes with a 'bet' that stakes currency. The VRF is based on the
  prev's block proven randomness as seed.
- Each block comes with a stake 'bet' that determines a threshold value that the
  vrf token needs to exceed
- When the token is lower then the threshold the bet can be placed and the honest
  majority will take it into account when entering the next round.
- When entering the next round the honest majority rank all blocks in the previous
  round and (if the threshold allows it) will submit a block that build on the
  highest ranking block in the last round.
- It will also collect bets from all blocks that weren't the highest ranking but
  can be used to encode this in the longest chain this way.
- The stake in the collects can flow through the ancestory of the chain until
  the block that provided the inputs for those bets to give the chain weight.
- As such the heaviest chain becomes the ancestory that has the most provable
  collected stake betting on it.

# Problem: Threshold function without "Total" amount of stake?

- What about stake that isn't being used for betting? The current threshold function
  shows logically that if the stake that if a lot of stake is ideal the frequency
  of block that pass the threshold quickly becomes lower. Can we keep a moving
  average of the amount of stake that is used and adjust use this as the "total"
  stake? But we only see the stake of people that are betting. But maybe we can solve
  for T, or multiply the stake to approximate how much. The T can be consitently
  read from the chain.
  Instead, a threshold based on the density of winning tokens over the amount of
  rounds: Empty rounds should bring the average down
- Either we come up with a threshold based on the average token strength (like bitcoin)
  but how to use some-ones stake then?
- Or we figure out a way to determine the active "total" stake, but without registration  

# Economy
Each block contains a 'bet' and 'collects'. A proposer bet's on a block in the
previous round and collect other bets of siblings of its parent block. It should
also use this opportunity to collect its own bets with interest. Bets are transactions to
the treasury identity. Collects are transactions from the treasury. Bets consist
out of multiple inputs that together determine the stake. Each _collected_ bets
distributes its split value of inputs back in the ancestory to the block that
contained the collected bet's output. This gives weight and economic finality
to the ancestory and thus a certain branch.

- If your bet is not collected, you don't lose anything. (if someone is late to
  the party there is no punishment or reward)
- If your bet is collected and it voted on the right prev it is returned with a
  small coinbase reward. (prevent witholding, incentivise sending in)
- If your bet is collected and it voted on the wrong parent it is returned without
  slashing it, but without any reward also. This should prevent incenstive for
  winning bets to be send in _just_ too late such that a large portion of the
  network loses a lot of money as they're voting on the wrong parent. Voting on
  the wrong parent can happen due to technical difficulties, slow network etc.
- The bet of the prev block must always be collected for a block to be valid such
  that the winning bet always gets rewarded.  
- If a double bet (vote) is collected the amount is never returned to the PK or
  is otherwise severly penalized
- The winning block gets a reward for each collected bet as coinbase. Collecting
  more bets is peferable.
- The treasury PK can not propose a block.

# Implementation

- What prevents members from choosing too quickly? You only get
- What if the ticket only determines if you'r added to the shortlist and everyone
  just picks the first it sees?

There are some implementation questions:
- How long does a member wait on other bets for a tip to come in? Honest majority members
  will use a timeout that is extended whenever a higher ranking block comes along. But what
  if other don't and propose a block for the next height? Don't accept it yet
- If the timer expires without any bet being placed. Each member can place another
  bet with the nonce increased by 1. Spamming multiple bets with many nonces will
  get you sliced.
- Sending in the highest nonce that passes the members threshold is a valid strategy
  but it will be ignored by members if it is too far of the current nonce.




- Each PK requires at least one stake in order to submit a block
- A block submission comes with verifiable random token that ranks it amongst siblings
- The token is based on the prev token (which is also provably random) and a nonce
- The nonce can be increased indefinitely until a lower was seen for a prev

## V2 Attack solution
- _VRF key grinding._; In which a member tries to generate a PK to get better VRFs.
  If it tries to do this for a specific block it will not be able to use it since
  the stake cannot be locked up yet. If it tries to do this longer term it is
  impossible since every block has new randomness that cannot be predicted.
_ _Nothing-at-Stake_: In which members vote on every tip. Since the nr of messages
  in the system are limited we can simply keep a black list for every height such
  that each PK can only submit one block per height. Alternatively, we can keep
  track from which child a block deep in the chain has gained it's stake and if
  it sees the a ticket at the same height from two different children it has proof
  of double voting.  
- _Nonce spam_ In which whenever a new tip is present, malicious members can decide to mine
  many new tips with different nonces and send them to the network. Each member only
  accepts nonce increased blocks after a timeout (and one above it). Everything else
  is dropped.
- _Long Range attack_ In which an alternative history is written that is also
  the "longest". We select the chain that had the most stake being bet on, this
  "weight" is encoded in each members chain. Taking over the main chain would
  require the attacker to put up more stake then the   



- The ticket is based on the Prev ticket and the nonce.
-

- Every block can be submitted with a slot that each member can choice.















# Wall
Beyond the need for a synchronised clock. The core utility of the rounds is that
it should force member to wait for all blocks to arrive and make an informed
decision.

- Idea: Some wall that blocks any block submissions from  a member after it has
  submitted one for a certain amount of time. Possibly powered by a bloom filter
- Idea: Do we still need it with the threshold function? We now have vrf that only
  allows a single token per round. It is already a bet
- Idea: Once a block has been broadcased keep a wall that prevents members from
  sending blocks that build on it? Or the ones before it.

# Questions
- how about old blocks that come in?
- What happens when a member sends multiple block proposals with different tokens
  per round? Do we need a firewall anyway? Currently it seems to be totally possible
  to send blocks with different hashes but the same token to the network. How are
  they sorted? For example, use the same token to send a block for every block
  at a certain height. Selfish mining?

# Bloom wall
A bloom filter (or Cuckoo Filter) can return whether a piece of data is definitely
not in a set or maybe in a set. When the filter returns 'definitely not in set' we
can accept its proposal. If the filter returns a maybe, we immediately block  the
proposal. In case of a false positive no harm is done since the VRF doesn't
consume resources for the minter.

 "The trade-off for this one-sided error is space-efficiency. Cuckoo Filters and Bloom Filters require approximately 7 bits per entry at 3% FPP, regardless of the size of the entries"
 ~ https://bdupras.github.io/filter-tutorial/

Building on the wrong block should bring opportunity costs, just like when a
PoW chain decides to work on the wrong tip. So if a member is only allowed to
make a block per N heights it should think carefully as it might miss out on
the reward that it was able to gain from building on the correct chain.

 "Create explanation of the nothing at stake problem:"
 https://www.mangoresearch.co/casper-nothing-at-stake-problem/

Alternatively (as explained here: https://github.com/ethereum/wiki/wiki/Proof-of-Stake-FAQ#what-is-the-nothing-at-stake-problem-and-how-can-it-be-fixed). We could encode the stake that someone
is willing to put up on a certain block. And return it to him if its the longest
chain. The threshold function ensures that spreading stake out doesn't increase
the chance and only one can win to this is a loss.

- Encode stake in block proposal, use the threshold function with the total stake
  in the system to determine the threshold
- This encoded stake must be less then the stake the user owns and take into account
  anything that is locked in the stake
- Give back the stake with a certain profit, this profit needs to be less then the
  stake.
- Randomnes of token can be based on the prev. Staking many blocks on old prevs
  just throws away stake.


- Problem: But how do we count stake reliably if it is not in the longest chain?
  the stake can be signed separately and then used as proof to slash it?
- Problem: Why not prevent nothing at stake altogether? At every height just allow
  one attempt per user. And how do we advise members to


- What if we stake, and block. If the block becomes part of the longest chain the
  stake is returned. Any member can later use the signed stake as a proof that the
  member double voted. by including it in a block. But what is double voting?
  At a certain height, you can only vote once.   


- 1) The VRF seed is purely based on the prev's block token, provably random
- 2) The threshold function limits the nr of follow up blocks that the network
     can submit to only a handfull.
- 3) The VRF pk cannot be grinded into getting a high token by making the PK a
     proof of work.
- 4) The stake a user puts into a block is signed and send together with the
     block. It can be used to verify by everyone that the threshold was reached.
- 5) Everyone keeps a blacklist such that each PK can only be used once per
     chain height.
- 6) Splitting the stake up into multiple identities to vote on multiple tips
     splits up the identity

1. If we base the token on the prev block alone it is unclear what should happen
   if no token arrives below the threshold. In a normal PoW this
2. If the token is based on a round or slot the members can move past that slot
   and a new slot nr becomes available to generate blocks for.


PROBLEM: What if no block is found? for a vrf?

- when do they commit to the token pk? how to prevent pk grinding? PK is a proof
  of work, worth x amount of bitcoin?
- use the prev as the randomness value, this unpredictable and as long as generating
  a pk takes longer then a normal round. It is not worth to generate a new pk every new block.
- the PK is used to send the funds back too. It must have some stake
- If anyone sees a stake message



## What to block on?
- Idea 1: Block the VRF PK, the pk cannot be used for a certain amount of time.
  this simulates a proof-of-work in that it takes time for a certain identity to
  generate a proof again. Except it would be fixed, or can it be random?
- Idea 2: block the vrf token itself. This just prevents the identity from mis-using
  a winning token to submit many blocks.
- Idea 3: block the signature PK

## Questions
- Is the vrf false-positive consistent/deterministic across members? If not this
  may cause some members to accept it, while others block it. Maybe it can be
  done on chain?
- What if the member is powerful and can send one block to one side of the
  network and one to the other?
- What prevents members from putting all PK's of all members on the blacklist? If
  the minority does this it might miss out on blocks that the majority is building
  on. If the majority does it the member is effectively prevented from minting.
- What if members decide not to put some PK's on the blacklist? If the minority
  does it they might receive more blocks to build on but the majority would not
  accept them. _But what about indirect blocks?_ We might need to keep filter
  over time, per height.
