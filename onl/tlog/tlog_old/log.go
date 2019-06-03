package tlog

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

// Proof is a serializable data structure that takes the hash of a record and
// returns the root hash of the log
type Proof [][2][sha256.Size]byte

// Log implements a transparent log. It is append only but hard to modify and
// requires no trust from the client. The main source of inspiration has been
// Russ Cox's blog post here: https://research.swtch.com/tlog but it also draws
// inspiration from Merkle Mountain ranges as descried here:
// https://github.com/mimblewimble/grin/blob/master/doc/mmr.md which resulted
// in the following reference implementation: https://github.com/zmitton/merkle-mountain-range
// It seems the flyClient recognized the importance of this: https://eprint.iacr.org/2019/226.pdf
type Log struct {
	dir  string
	recf *os.File
	idxf *os.File
	lvlf []*os.File
	mu   sync.RWMutex
}

// NewLog creates a new transparent log
func NewLog() (l *Log) {
	l = &Log{}
	var err error

	l.dir, err = ioutil.TempDir("", "tlog_")
	if err != nil {
		panic(err)
	}

	l.recf, err = os.OpenFile(filepath.Join(l.dir, "records"), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}

	l.idxf, err = os.OpenFile(filepath.Join(l.dir, "index"), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}

	return l
}

func (l *Log) Read(n uint64) (r []byte) {
	if n < 1 {
		return nil //no record 0 exists, @TODO maybe we can use it return configuration info?
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	var (
		bpos uint64
		epos uint64
	)

	b := make([]byte, 16)
	rn, err := l.idxf.ReadAt(b, int64((n-1)*8))
	if err == nil {
		//middle record, read until beginning of next record
		bpos = binary.BigEndian.Uint64(b[:8])
		epos = binary.BigEndian.Uint64(b[8:])
	} else {

		// if nothing was read the error is unexpected. If something was read we
		// read EOF because it was the last record
		if rn != 8 {
			panic(err)
		}

		//read until end
		bpos = binary.BigEndian.Uint64(b[:8])
		eofp, _ := l.recf.Seek(0, 1)
		epos = uint64(eofp)
	}

	r = make([]byte, epos-bpos)
	_, err = l.recf.ReadAt(r, int64(bpos))
	if err != nil {
		panic(err)
	}

	return
}

func (l *Log) openLvlF(lvl int) (f *os.File) {
	if len(l.lvlf) < (lvl + 1) {
		var err error
		f, err = os.OpenFile(filepath.Join(l.dir, strconv.Itoa(lvl)), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			panic(err)
		}

		l.lvlf = append(l.lvlf, f)
	}

	f = l.lvlf[lvl]
	return
}

// hash data at a certain level into a file.
func (l *Log) hashToFile(d []byte, lvl int) {
	f := l.openLvlF(lvl)
	h := sha256.Sum256(d)
	_, err := f.Write(h[:])
	if err != nil {
		panic(err)
	}

	newlen, err := f.Seek(0, 1)
	if err != nil {
		panic(err)
	}

	// check if appending resulted into a new pair of hashes
	if (newlen/sha256.Size)%2 == 0 {

		// if so, read these two child hashes from
		children := make([]byte, 2*sha256.Size)
		_, err := f.ReadAt(children, newlen-(2*sha256.Size))
		if err != nil {
			panic(err)
		}

		// and then hash them together and add the parent one level up
		l.hashToFile(children, lvl+1)
	}
}

// Stat returns the current top level hash and the number of records stored
func (l *Log) Stat() (t [sha256.Size]byte, n uint64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// how can we, given the size determine which levels and offsets to read?
	// and how would that specifically look like for sparse trees?
	// how to calculate ephemeral hashes

	// calculate top level hash and the nr

	return
}

// lasth returns the last hash at every level
func (l *Log) lvllen(L int) int64 {
	f := l.openLvlF(L)
	len, err := f.Seek(0, 1)
	if err != nil {
		panic(err)
	}

	if len == 0 {
		panic("lvl file without any hashes")
	}

	// _, err = f.ReadAt(h[:], len-sha256.Size)
	// if err != nil && err != io.EOF {
	// 	panic(err)
	// }

	return len / sha256.Size
}

// Return hash nr K at level L
func (l *Log) h(L int, K, n uint64) (h [sha256.Size]byte, ok bool) {

	f := l.openLvlF(L)

	_, err := f.ReadAt(h[:], int64(K*sha256.Size))
	if err != nil && err != io.EOF {
		panic(err)
	} else if err == io.EOF {

		// we can decompose and then look at the largest power of 2. Then determine
		// the max height by

		// given the size of the tree we should be able to determine:
		// - the max amount of levels.
		//  - a log with max uint64 records only needs 64 levels. A proof is at most 4kb
		//  -
		// - whether a given level+offset is actually a empheral hash
		//   created from hashes on a different level

		// could be that we reached the top.
		// - Can we determine how many levels a log n records has?
		// could be that we need to reach into ephemeral hashes

	}

	return h, true
}

// Contains returns a proof that the log of size 'n' contains a record number 'i'.
func (l *Log) Contains(i, n uint64) (p Proof) {

	// decompose into perfect binary trees to make assertions about the shape
	decomp := pow2decompose(n)
	if len(decomp) < 1 {

		panic("fix me")
	}

	var peaks [][sha256.Size]byte
	for _, nn := range decomp {

		tiplvl := log2(nn)
		tipi := l.lvllen(tiplvl)
		tiph, ok := l.h(tiplvl, uint64(tipi-1), n)
		if !ok {
			panic("no tip hash of perfect tree, should not happen")
		}

		peaks = append(peaks, tiph)

		// build the proof for each decomposed tree

		// keep x, as the right sibling while building up.

		// there is an ephemeral hash combining each peak. At a level one above
		// the highest of the two peaks.

		// fmt.Printf("tree(n=%d) peak h(L=%2d, k=%2d) = %.4x\n", nn, tiplvl, tipi, tiph)
	}

	fmt.Println(decomp)
	fmt.Println(peaks)

	// the max level is determines by the height of the largest
	// decomposed perfect binary tree.
	maxlvl := log2(decomp[len(decomp)-1]) + 1

	width := n
	for L := 0; L <= maxlvl; L++ {

		//which sibling do we read for this level
		if i%2 == 0 {
			i++ //even, get right sibling
		} else {
			i-- //odd, get left sibling
		}

		//try to read the sibling
		h, _ := l.h(L, i, n)

		// @TODO how to determine max width per level from composition?
		// for each decomposed tree
		// fmt.Printf("L=%d, width=%d, i=%d\n", L, width, i)
		if i >= width {

			//we instead, add the tip of the rightmost component
			h = peaks[0]
			peaks = peaks[1:]

			fmt.Println("\t@", L)
		}

		fmt.Printf("h(L=%d, K=%d) = %.4x\n", L, i, h)

		//add sibling hash to the pair
		var pair [2][sha256.Size]byte
		if i%2 == 0 {
			pair[0] = h //even, set left of pair
		} else {
			pair[1] = h //odd, set right of pair
		}

		//add pair to the proof
		p = append(p, pair)

		//at the next level, i is half of the current i
		i = i / 2

		// every level cuts the nr of
		width = width / 2
	}

	return
}

// Append record r to the log and returns its record number (starting at 1).
// The record number also represents the new size of the log.
func (l *Log) Append(r []byte) (n uint64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// capture the offset the record will be written at
	reco, err := l.recf.Seek(0, 1)
	if err != nil {
		panic(err)
	}

	// write the actual record
	_, err = l.recf.Write(r)
	if err != nil {
		panic(err)
	}

	// write the record offset to the index
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(reco))
	_, err = l.idxf.Write(b)
	if err != nil {
		panic(err)
	}

	// get the new index offset
	idxo, err := l.idxf.Seek(0, 1)
	if err != nil {
		panic(err)
	}

	// append hashes to the different level files
	l.hashToFile(r, 0)

	// use the new index size to return the number of the newly written record
	return (uint64(idxo) / 8)
}
