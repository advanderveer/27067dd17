# Random Beacon

N=5 // total group size
t=3 // threshold value
gpk // group public key ()

Rounds
- BLS to recover (create) signatures from a seed (previous round)
- There is only one valid signature (for a defined threshold)
- A defined group can recover the signature from a threshold size
- A group public key is used for signature verification for everyone
- The signature is a random number
- Seed+enough (t) shares gives everyone a deterministic random number
- Each signature selets a subset of the network using a VRF
- Notarization is a time window that stays open for the highest block to arrive
  and then generates a threshold signature

- A round can be finalized by waiting on a T threshold signature on high
  ranking blocks. i.e we move to the next round once have received signature shares
  to verify the block   

## Resources
- BLS as VRF (Feb 2017): https://assets.ctfassets.net/ywqk17d3hsnp/6rxjadg91eWYmagOKQe0cS/d63dc10003111533c5632552b44ccb78/threshold-relay-blockchain-stanford.pdf
- Randomness Compared: https://medium.com/mechanism-labs/randomness-in-blockchains-part-1-79192b173816
- Ouroboras VRF usage: https://medium.com/unraveling-the-ouroboros/introduction-to-ouroboros-1c2324912193
