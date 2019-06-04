# Burn
Burn is a proof-of-burn blockchain that uses a VRF to draw random tickets in a
slot based system.

## Economy and Consensus

 - There are three kinds of transfers: Normal, Coinbase and Burn
 - Burn allows anyone to buy a virtual mining rig.
 - We assume a good portion of those virtual rigs will be participating.
 - We will progress in rounds.  

 - What if old outputs are discarded, only wat to have long term investment
   is through active participation? Having a ever larger virtual mining rig?

## The Barriers of running a full node

### Problem 1 - UTXO memory size
UTXO Size is getting large, according to Peter Todd this is a problem because
(https://petertodd.org/2016/delayed-txo-commitments) competitive mining (participation)
requires this set to be in-memory. It also overall increases the barrier to entry.

 - For our setup this is less of a problem because it's not a race. The utxo set
   can reside on disk. Possibly with a cuckoo filter in front.
 - we do care about the accessibility issue. We don't want people to have to
   reserve several gigabytes of memory in order to participate; but reading it
   this amount from disk would be ok - although, not ideal.

### Problem 2 - Chain history size  
In order to participate with a full node, it is necessary to download the full
chain history. Currently this is 150+ GB for a whole sync, which puts too much
strain on the participant.

 - Downloading the full history should not take this much time. Or may not be
   necessary at all?
 - We would like to have snapshots being uploaded and signed by members to
   some mirror server that can cache or server them locally for other clients.
 - Snapshot may be re-created from the chain history log.

### Problem 3 - The unbounded ness of the the above two problems
Both the bootstrapping size and the UTXO set are growing unbounded. There are no
rules in place that will cause them ever to shrink or remain constant. Although it
is a long-term problem it almost certainly require rules that are fundamental to
how the chain works.

- _Idea 1:_ what if every transfer you make to yourself can only be to one output?
  - can we make a UTXO system that maintains an average of less then 1 "child"
    transfers. Such a graph is guaranteed to go extict. Like you can have 5
    outputs to yourself, but you must burn at least 5 in the process.  
- _Idea 2:_ different types of data, archived vs active data stored differently. Some info
  becomes read only, or very expensive to write/update?
- _Idea 3:_ we reach consensus over a status oracle in which every identity has
  a nr of keys proportional to their stake? But at some point, keys become unreadable
  as part of a write transaction? data needs to be unarchived into a pure write
  transaction to make it available to the oracle again. The status oracle size
  should be bounded.

### Design V6

- We keep a verifiable log of TxData: key writes and commit/start timestamps:
  https://research.swtch.com/tlog
- This log can be hosted and cached efficiently and forms the unchangable log
- The consensus protocol reaches a conclusion on the tip of the log, that becomes harder
  to change over time.
- At some point the chain can "flush" to the permanent log.
- The log can be used as an input to the chain.
- The log can easily be archived and uploaded to a large storage at any height  
  and anyone can read from it efficiently.
- A fixed sized status oracle is kept at the tip of the chain.
- Tip reorganization should be easy with the log stored locally.
- The status oracle only accepts read-then-write transactions that read a key
  that was recently committed.
- If an older key needs to be updated it needs to be unarchived by the proposer
  and written by hand.
- In this case there is not atomic guarantees for this data, others might overwrite
  it concurrently.


### Technologies / Algorithms

- sharding/zk-snarks: https://petertodd.org/2015/why-scaling-bitcoin-with-sharding-is-very-hard
- txo commitments MMR: https://petertodd.org/2016/delayed-txo-commitments
- utxo growth problem: https://eklitzke.org/an-overview-of-bitcoin-utxos
- rolling utxo set hashes: https://lists.linuxfoundation.org/pipermail/bitcoin-dev/2017-May/014337.html
  https://github.com/bitcoin/bitcoin/pull/10434
- Merkle logs: https://research.swtch.com/tlog - Has some nice research the use of
  tiling such that the server can be considered untrusted. To store or mirror
  the complete log of the block chain efficiently
- homomorphic hashing: https://github.com/lukechampine/lthash to iteratively
  hash the UTXO set. UTXO commitment
- bloom/cuckoo filters: https://brilliant.org/wiki/cuckoo-filter/
- Accumulators: https://ethresear.ch/t/accumulators-scalability-of-utxo-blockchains-and-data-availability/176/27
