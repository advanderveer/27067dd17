package bchain

// DB provides the blockchain with a transactional data store
type DB interface {
	Update(func(tx Tx) error) error
	View(func(tx Tx) error) error
	Close() error
	Destroy() error
}

// Tx describes the database transaction
type Tx interface {

	// ReadTip reads the current tip from storage
	ReadTip() (id BID, w uint64, err error)

	// WriteTip writes a new tip as the current tip with its weight to storage
	WriteTip(id BID, w uint64) (err error)

	// ReadWeight will read a blocks weight by its token
	ReadWeight(id BID) (w uint64)

	// WriteWeight will write a blocks eight by its token
	WriteWeight(id BID, w uint64) (err error)

	// Read just the block hdr and consensus weight
	ReadBlockHdr(id BID) (b *BlockHdr, err error)

	// Write a block to storage, replacing any existing data
	WriteBlock(b *Block) (err error)

	// Iterate over all round from start to end and call f for each
	EachBlock(start, end uint64, f func(n uint64, hdrs []*BlockHdr) error) (err error)
}
