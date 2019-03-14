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
  - [ ] test the cycle with 2 members
- [ ] figure out how to protect against grinding attack that tries out many old
      blocks as 'pref' to find a very high draw
- [ ] figure out how to protect against a halting of the system (coin death?)

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
