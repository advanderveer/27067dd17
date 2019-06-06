package simulator

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/big"
)

//ID uniquely identifies a block
type ID [32]byte

//NilID is an empty ID
var NilID = ID{}

//Block in the blockchain
type Block struct {
	priol  uint8  //256 should be enough
	prio   []byte //capped at 32 right now
	ticket [32]byte
	proof  [96]byte
	widx   uint64

	//@TODO add more possible variation then 8 bytes? https://bitcoin.stackexchange.com/questions/17327/is-it-possible-to-run-out-of-nonce-values
	nonce uint64
	prev  ID
	//@TODO add cumulative weight/strength to allow the receiver to easily detect a new tip
	//@TODO store difficulty? But we can already have prio? https://en.bitcoin.it/wiki/Difficulty
	//@TODO add height, usefull for filtering
	//@TODO VRF public key for ticket verification(?)
	//@TODO signature to protect from tampering
	//@TODO actual playload data?
	//@TODO about difficulty and timestamps: https://medium.com/@g.andrew.stone/the-blockchain-difficulty-information-paradox-879b0336864f
	//@TODO mining difficulty calculations: https://medium.com/chainalysis/bitcoin-mining-calculations-ff90dc958dba
}

//NewBlock will allocate a fixed size block
func NewBlock(prio *big.Int, widx uint64, ticket, proof []byte, prev ID) (b *Block) {
	b = &Block{
		widx: widx,
		prev: prev,
	}

	priob := prio.Bytes()
	b.priol = uint8(len(priob))
	if b.priol > 32 {
		panic(fmt.Sprintf("priority is a number larger then fits into a 32 byte field, it is '%d' bytes", b.priol))
	}

	b.prio = make([]byte, b.priol)

	copy(b.prio[:], prio.Bytes())
	copy(b.ticket[:], ticket)
	copy(b.proof[:], proof)

	return
}

// DecodeBlock decodes a block from reader 'r' and return it
func DecodeBlock(r io.Reader) (b *Block, err error) {
	b = &Block{}
	for _, v := range []interface{}{
		&b.priol,
		&b.prio,
		&b.ticket,
		&b.proof,
		&b.widx,
		&b.nonce,
		&b.prev,
	} {
		if b.priol > 32 {
			return nil, fmt.Errorf("priority length is too big, got: %d", b.priol)
		}

		if uint8(len(b.prio)) != b.priol {
			b.prio = make([]byte, b.priol)
		}

		err := binary.Read(r, binary.LittleEndian, v)
		if err != nil {
			return nil, fmt.Errorf("failed to write binary data: %v", err)
		}
	}

	return
}

//Prio returns the block priority as drawn in the lottery
func (b *Block) Prio() (prio *big.Int) {
	prio = new(big.Int)
	prio.SetBytes(b.prio[:])
	return
}

// Encode the block to the provide writer
func (b *Block) Encode(w io.Writer) (err error) {
	for _, v := range []interface{}{
		b.priol,
		b.prio,
		b.ticket,
		b.proof,
		b.widx,
		b.nonce,
		b.prev,
	} {
		err := binary.Write(w, binary.LittleEndian, v)
		if err != nil {
			return fmt.Errorf("failed to write binary data: %v", err)
		}
	}

	return
}

// Hash the block bytes and return it as a Block's ID
func (b *Block) Hash() (id ID, err error) {
	sha := sha256.New()
	err = b.Encode(sha)
	if err != nil {
		return NilID, fmt.Errorf("failed to encode block: %v", err)
	}

	copy(id[:], sha.Sum(nil))
	return
}

// Difficulty deterministically returns the mining difficulty in the range between
// minD and maxD based on the drawn priority. A higher priority returns a lower
// difficulty that needs to be mined
func (b *Block) Difficulty(maxPrio *big.Int, minD, maxD int64) (d uint) {
	if maxD > (sha256.Size * 8) {
		panic("max difficulty cannot be bigger then number of bits in hash")
	}

	//@TODO can we enforce a difficulty ajustment based on past block information

	//@TODO the difficulty (nBits/bits in bitcoin) also functions as an indicator
	//for chain strength. And is therefore important for the consensus mechanism
	//bitcoin sums up the the difficulty to get at the chain with the most strenght
	//but we cannot directly do that. As in our case, we compare based on prio.
	//idea: map large int prio to a value in int8 space and return that as a
	//weight/strength indicator. which can be added up?

	//@TODO the difficulty can probably also be calculated by subtracting
	//the drawn prio from the max prio and using that as a threshold.

	frac := new(big.Rat).SetFrac(b.Prio(), maxPrio) //normalize prio range 0..1
	frac.Sub(new(big.Rat).SetInt64(1), frac)

	frac.Mul(frac, new(big.Rat).SetInt64(maxD-minD)) //multiply into range
	frac.Add(frac, new(big.Rat).SetInt64(minD))      //add min difficulty into minD..maxD

	f, _ := frac.Float64()
	return uint(f) //simply truncate (@TODO how portable is that)
}

// Mine the block, it will set the nonce to a value such that the hash of the
// block is lower then the max value minus the difficulty.
func (b *Block) Mine(maxPrio *big.Int, minD, maxD int64) (pow ID, err error) {
	difficulty := b.Difficulty(maxPrio, minD, maxD)
	proofi := big.NewInt(0)
	target := new(big.Int).Lsh(big.NewInt(1), (sha256.Size*8)-difficulty)

	//@TODO figure out what it means that all fields are only protected by the PoW
	//itself. The PoW difficutly is protected by the ticket that and is hashed with it
	//so what can one change with a ticket and low difficulty? send many blocks with
	//different data? How to make sure a ticket can only we be used once?

	for b.nonce = uint64(0); b.nonce < math.MaxInt64; b.nonce++ {
		pow, err = b.Hash()
		if err != nil {
			return NilID, fmt.Errorf("failed to hash: %v", err)
		}

		if proofi.SetBytes(pow[:]).Cmp(target) == -1 {
			break
		}
	}

	return
}
