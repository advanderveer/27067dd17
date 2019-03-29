# Two Tower Concept
A blockchain that chains two kinds of blocks

1) _Voters_ broadcast votes on a block (tip). They may be selected through a VRF lottery and
need to show their token as proof together with the block ID they vote on. Each vote
refers to a single other block in this way. They pick the strongest tip to vote on,
this naturally ensures that stronger tips will have more votes.

2) _Talliers_ gather votes for different tips until the can solve a math puzzle
using any combination of these votes. When they do they can make a new block with
the prev block of the votes they gathered. All members can quickly verify this and
add the block to the chain. Talliers prefer making proofs with with votes on the
strongest chain as this increase the chance of their block being included in the chain

Theory:
- Voters will always make votes for any tip that is longer then the previous tip they
saw
- Voters will keep doing this until a new tip arrives. Voters are always working on
one tip.
- Talliers will be happy to make proofs for any set of votes it gathers.

3) Whenever a tally is received members will verify its proof and add the block to
their chain.

## Questions:
- _Q: Why have two roles?_ A: Whenever there is a mechanism that combines multiple  
messages into one there needs another mechanism that creates more messages. Else
the system will have an ever decreasing amount of messages.  
- _Q: What happens if no-one can find a mathematical proof?_ A: Proposers can send
multiple proposals (with different) tokens after they solve a PoW puzzle. Talliers
then have more material to create a proof.
_ _Q: Store all proposals in the chain?_ A: There might be many more proposals
then there are blocks, we would like to not store them at all for. This is a reason
for letting the talliers chose the block. Voters offer material for a tip.
_ _Q: Proposers can propose/grind on any tip? Why not propose on any tip? Majority will
not do this?_
_ _Q: How to prevent voters creating infinite votes for an old tip_ have a proof of
work for casting votes. the nr of votes for later tips will outperform old tips that
is voted on
- _Q: How to increase finality?_

## Ideas:
- Send votes over UDP.


# Beacon concept
Voters draw a vrf based on the tip. they solve a PoW to send a vote every N seconds
on average. Everyone can gather distict proof votes together to propose a new proof. This
proof of work doesn't increase with the nr of members, its fixed and solving it shouldn't
increase the reward. The work in the system is configured through the number vrf threshold
so it should be constant. Talliers can only combine votes from distinct voters, adding more
machines to flood the network with votes doesn't increase the reward.
PRO: mechanism to create more votes, less roles and logic
PRO: have an ensurance that the system never halts
CON: slow? but we need some rating mechanism anyway.

1) Voters send votes for a certain tip over the network. Each vote requires
a small proof of work (PoW) to prevent spamming. This PoW's difficulty doesn't change
as the the network increases in size. It is purely as a rate limiting mechanism. It contains
a fixed amount of data and can be broadcasted over UDP, losing one is not a problem.
With a VRF the message load for the network of this voting beacon is constant.

2) Miners combine votes from distinct voters such that it solves a mathematical function.
When this puzzle is solved anyone (or VRF based) can broadcast a new block of data. Anyone
can quickly verify that the proof is correct as the votes are embedded in the message.
The difficulty  

3) Every member only accepts and relays votes for the (height?) of the tip they are at and
switches once a new tip comes along. To offer finality.

## Components and Mechanims
- Short Proof of Work for VRF token vote @~1 sec, tip height determines the token
- Combine votes from distict voters to solve another math puzzle, if solved closes the height
- Relay policy only allows for relaying votes for.

## Problems / Questions
- _How to prevent one person generating a lot of votes for a miner, or how to make
sure that throwing more computers at it doesn't work?_
- _Why would anyone broadcast votes to the network?_ there is a reward for being included
in a block.
- _But then how to NOT reward spamming?_ Network filters multiple votes for a tip(?)
- _But then how to prevent people generating there own votes?_ A block requires votes from
distinct voters so selfish people will be outperformed.
- _But then how to prevent chain death from lack of votes?_

- _How to prevent people from emitting votes for each tip?_ Add finalization mechanism
small spam proof of work

## Dilemma
Either allow for a mechanism that allows continuing after no block was found or
allow each identity to create multiple votes per tip. Or can we balance the difficulty
and VRF thresholds such that it just very unlikely that no-one finds a solution?
Or simply remove any incentive of spamming the network with votes? Only speeds up
the algorithm?
