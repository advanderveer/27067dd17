# Decentralised Transactional Key-Value Store
[introduction to the state of the world]
[we want to be in control, but it has to be convenient]
[decentralized data manipulation but (encrypted) centralized storage]

# Technologies and Research
[WebRTC for p2p communication]
[Proof-of-Stake for efficient consensus]
[Serializable Snapshot Isolation for ACID data manipulation]
[Transparent Log for data archival and tiering]

# Big ideas and Scalability
- _Bound status Oracle_ we only provide serializable data manipulation on recent data, giving an upper
  bound to the required memory needs to works with. Old data should be readable
  and verifiable from untrusted archive mirrors but can only be used again after
  deliberate writing into the recent-data view.
- _Log(n) history verification_ recent consensus is still build on old archived state
  that should be easy to verify and continue-on without downloading gigabytes of
  data.
- _VRF Thresholds_ the network needs to be infinitely scalable in terms of the
  nr of messages. With a VRF based consensus it is possible to create a fixed
  upper bound to the messages no matter how many members are participating.


# Transaction Lifecycle
- clients formulate the transaction by reading certain rows from a untrusted storage server.
  the server returns just the latest version and a proof that that data is in the log.
- clients can operate by reading their encrypted data an re-encrypting it to write
  data to the service.
- each proposer (miner) doesn't need to know about the data, it just keeps a status oracle.
  it validates the proof against the latest treehash and size it has. it keeps this
  by watching the storage server.

  It additionally needs to check that the the rows was not written after this transaction read it.
  check that transactions only write to the keyspace of the signer of the transaction.
  After this succeeds it can append transactions to blocks and propose it.
- each proposer validates incoming blocks against its tree hash and prefix
  If the block is valid and becomes the longest chain it is re-broadcasted and
  new tip can be build on.
- The storage server also watches the chain and whenever a new longest chain is
  present it will revert the value log and append all new entries to the log.

- each role watches the blockchain and whenever a new tip becomes the heaviest
  it will switch its log around and append data that is relevant for itself to
  the new log.

# Operations
- storage.ReadRows(rr []keys) ([]rows, ps []proof) //read the latest row data, return proofs the data exist  
- network.SubmitTx(ts int64, wr, rr []row) //broadcast, ask proposers to mine block for it 

# Ideas / Questions
- [ ] Can we use the tree proof to make it possible for members to join without them
  needing to download the entire block history. Just a proof that it exists?
  - basically start receiving blocks and building a chain halfway?
- [ ] How would a fixed sized status oracle looks like that operates on transactions
  that read at a certain block and write rows at a certain block.  
- [ ] Or can the merkle log be used for efficient block syncing?
- [ ] If a commitment hash is included how can the recent data state be synced?
- [ ] we can instead add all transactions to the log instead of the blocks?
- [ ] Each block can come with a proof that when the transactions are applied
      the prev tree is a prefix to the new tree. With just the new size, the new
      hash and the prev size and hash.
- [ ] For transaction validation we just need to keep the last commit hash of the
      block the row was written at. trying to read a value that is older then
      a certain depth will always fail. status oracle commit data that is older
      then that can always be discarded.
- [ ] For actual transaction formulation we need to be able to read certain row
      data at some version allow filling the the write rows. Can this be done
      from a trustless server that shows that the row is indeed in the log
- [ ] what if each transaction instead comes with a merkle proof that
      can be verified against a server?
      - a transaction comes with "read-block id" which is the height at which it
        reads all its rows.
      - its needs to show that no other transaction wrote to those rows rows after
        that time...
      - at the time the block is formulated and verified.
      - can this be done with multiple untrusted proving servers?
      - can the transparent log be turned into a dag with deduplicated underlying
        storage?
- [x] On switching tips can we rollback a transparent log and initate a new one? _yes_


# Data structures
- Block Chain - Keeps all blocks that the the node received in order mapped to
  the round they appeared and keep the longest chain as the tip it builds on.

- Status Oracle - Recent data that can be manipulated with ACID transactions.
  Based on the tip of the chain it has the version of most recent commits. With
  this info new transactions and can be validated. It maps a fixed nr of keys per
  identity to the height it was committed at. If this height becomes to far into
  the past compared to the current height it can no longer be read.

- Transparent Log - The longest chain gets stored in a transparent log that can
  generate very efficient proves that another chain prefixes it. It can also
  be used to audit and rebuild the recent-data view.
