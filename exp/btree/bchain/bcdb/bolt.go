package bcdb

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/advanderveer/27067dd17/exp/btree/bchain"
	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
)

var (
	boltBucketBlocks  = []byte{0x00}
	boltBucketTips    = []byte{0x01}
	boltBucketWeights = []byte{0x02}
)

// Bolt is a blockchain database implementation that uses the Bolt db
type Bolt struct {
	dir string
	bdb *bolt.DB
}

// MustTempBoltDB will create a temporary Bolt database
func MustTempBoltDB() *Bolt {
	tmpd, err := ioutil.TempDir("", "bchain_tmp_")
	if err != nil {
		panic("bcdb/bolt: " + err.Error())
	}

	b, err := NewBoltDB(tmpd)
	if err != nil {
		panic("bcdb/bolt: " + err.Error())
	}

	return b
}

// NewBoltDB will initialize a bolt database, the directory must exist
func NewBoltDB(dir string) (b *Bolt, err error) {
	b = &Bolt{dir: dir}

	//open or create the database file
	b.bdb, err = bolt.Open(filepath.Join(dir, "bchain.bolt"), 0600, nil)
	if err != nil {
		return nil, errors.Wrap(err, "bcdb/bolt: failed to open or create database file")
	}

	if err = b.bdb.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(boltBucketBlocks)
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists(boltBucketTips)
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists(boltBucketWeights)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "bcdb/bolt: failed to create block bucket")
	}

	return
}

// Update the database
func (db *Bolt) Update(f func(tx bchain.Tx) error) (err error) {
	err = db.bdb.Update(func(tx *bolt.Tx) error {
		return f(&boltTx{tx})
	})

	if err != nil {
		return errors.Wrap(err, "bcdb/bolt: update failed")
	}

	return
}

// View data in a read-only transaction
func (db *Bolt) View(f func(tx bchain.Tx) error) (err error) {
	err = db.bdb.View(func(tx *bolt.Tx) error {
		return f(&boltTx{tx})
	})

	if err != nil {
		return errors.Wrap(err, "bcdb/bolt: view failed")
	}

	return
}

// Close the database file
func (db *Bolt) Close() error {
	return errors.Wrap(db.bdb.Close(), "bcdb/bolt: failed to close")
}

// Destroy the database file and all its contents
func (db *Bolt) Destroy() error {
	return errors.Wrap(os.RemoveAll(db.dir), "bcdb/bolt: failed to destroy")
}

// boltTx is the concrete transaction implementation for a bchain tx
type boltTx struct {
	btx *bolt.Tx
}

// ReadWeight will read a blocks weight by its token
func (tx *boltTx) ReadWeight(id bchain.BID) (w uint64) {
	data := tx.btx.Bucket(boltBucketWeights).Get(id.Bytes())
	if len(data) < 8 {
		return w
	}

	w = binary.BigEndian.Uint64(data[:])
	return
}

// WriteWeight will write a blocks eight by its token
func (tx *boltTx) WriteWeight(id bchain.BID, w uint64) (err error) {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, w)
	err = tx.btx.Bucket(boltBucketWeights).Put(id.Bytes(), data)
	if err != nil {
		return errors.Wrap(err, "failed to put weight")
	}

	return
}

// ReadTip will read the current tip from storage
func (tx *boltTx) ReadTip() (tip bchain.BID, w uint64, err error) {
	data := tx.btx.Bucket(boltBucketTips).Get([]byte{0x01})
	if len(data) < (len(tip) + 8) {
		return tip, w, bchain.ErrTipNotExist
	}

	copy(tip[:], data[:len(tip)])
	w = binary.BigEndian.Uint64(data[len(tip):])
	return
}

// WriteTip will write the current tip from storage
func (tx *boltTx) WriteTip(tip bchain.BID, w uint64) (err error) {
	data := make([]byte, len(tip)+8)
	copy(data, tip[:])
	binary.BigEndian.PutUint64(data[len(tip):], w)

	err = tx.btx.Bucket(boltBucketTips).Put([]byte{0x01}, data)
	if err != nil {
		return errors.Wrap(err, "failed to put tip")
	}

	return
}

// Read one specific block from storage
func (tx *boltTx) ReadBlockHdr(id bchain.BID) (bh *bchain.BlockHdr, err error) {
	data := tx.btx.Bucket(boltBucketBlocks).Get(id.Bytes())
	if data == nil {
		return bh, bchain.ErrBlockNotExist
	}

	return tx.decodeBlock(data)
}

// Write a block to storage unconditionally
func (tx *boltTx) WriteBlock(b *bchain.Block) (err error) {
	buf := bytes.NewBuffer(nil)

	enc := gob.NewEncoder(buf)
	err = enc.Encode(b.Header)
	if err != nil {
		return errors.Wrap(err, "failed to encode block header")
	}

	err = tx.btx.Bucket(boltBucketBlocks).Put(b.ID().Bytes(), buf.Bytes())
	if err != nil {
		return errors.Wrap(err, "failed to put block")
	}

	//@TODO write the block data
	return
}

// Walk all blocks from the starting round to the end round and call f
func (tx *boltTx) EachBlock(start, end uint64, f func(n uint64, hdrs []*bchain.BlockHdr) error) (err error) {
	c := tx.btx.Bucket(boltBucketBlocks).Cursor()
	if end < start {
		panic("reverse is not supported")
	}

	min := make([]byte, 8)
	bchain.WriteRound(start, min)
	max := bchain.NewBID(end).Bytes()
	for i := 8; i < len(max); i++ {
		max[i] = 0xff //fill with max possible key value
	}

	var curr uint64             //current round
	var hdrs []*bchain.BlockHdr //block headers
	for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
		bh, err := tx.decodeBlock(v)
		if err != nil {
			return err
		}

		round := bchain.ReadRound(k)

		//moving to next round, call f and reset round info
		if round > curr {
			err = f(curr, hdrs)
			if err != nil {
				return err
			}

			hdrs = make([]*bchain.BlockHdr, 0)
			curr = round
		}

		hdrs = append(hdrs, bh)
	}

	//call with any remaining round data
	err = f(curr, hdrs)
	if err != nil {
		return err
	}

	return
}

func (tx *boltTx) decodeBlock(data []byte) (bh *bchain.BlockHdr, err error) {
	dec := gob.NewDecoder(bytes.NewReader(data))
	bh = new(bchain.BlockHdr)
	err = dec.Decode(bh)
	if err != nil {
		return bh, errors.Wrap(err, "failed to decode block header")
	}

	return
}
