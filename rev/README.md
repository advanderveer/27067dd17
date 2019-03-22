

At the beginning of the round, each member draws a random value and if they
are past a certain threshold can propose a block to the network.

The random value also determines a mathematical puzzle that can be solved by
collecting a random sample of the network: the more you collect, the more
likely you are to to solve it. The vrf token is input to this puzzle to make
it different for every member of the network.

If anyone solves it, the member may broadcast the solution with a new block
and this will cause everyone to move to the new round. If multiple solutions
happen at the same time and members observe multiple solutions they pick the
one with the heaviest chain.

round start: draw a ticket from the highest observation of the previus round
or the genesis round.

proposal: contains the VRF token (based on round), and must contain a valid observation
observation: proofs that the proposer has waited by showing a list of proposals in the
previous round that hash together to solve the puzzle. The puzzle difficulty is low
and the hard part should be gathering the different observations. All listed proposals
are ordered by rank and should be lower then 'prev'.

The only way to solve the puzzle is to wait and collect enough evidence that shows
that you've ranked the proposals in the right order. If you rank the low part of the
proposals you'l run the risk of not being included in the chain are all because the weight
of your proposal is terrible and others will have seen a better one.

If during the observation collection a better tip comes along, we should switch to it

idea 1: draw the ticket based on the puzzle hash. This introduces randomness but incentives
the proposer to grind different observations.

idea 2: the vrf token is also based on the observation, this motivates grinding different
observation combinations and it not known when the process is done.

idea 3: if all combinations are exhaused member may start on a normal proof-of-work
to prevent chain death? This should be unlikely and the reward should be small enough
that it doesn't become the default mode of work.
