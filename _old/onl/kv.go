package onl

import (
	"bytes"
	"encoding/binary"

	"github.com/advanderveer/27067dd17/onl/ssi"
)

const (
	balanceKey = "_balance"
	stakeKey   = "_stake"
	tpkKey     = "_tpk"
)

//KV abstraction build on top of our block chain
type KV struct{ *ssi.Tx }

// DepositStake locks currency as stake and commits a token pk
func (kv *KV) DepositStake(owner PK, amount uint64, tpk []byte) {

	//read balance
	balk := append(owner[:], []byte(balanceKey)...)
	balv := kv.Get(balk)
	if len(balv) < 8 {
		return //no balance, do nothing
	}

	balance := binary.BigEndian.Uint64(balv)
	if balance < amount {
		return //nog enough balance, cannot do anything
	}

	//read current stake and add amount
	stakek := skey(owner)
	stakev := kv.Get(stakek)
	if len(stakev) >= 8 {
		return // if stake is already set, do nothing
	}

	//reduce balance
	balance -= amount
	binary.BigEndian.PutUint64(balv, balance)
	kv.Set(balk, balv)

	//set stake value
	stakev = make([]byte, 8)
	binary.BigEndian.PutUint64(stakev, amount)
	kv.Set(stakek, stakev)
	kv.Set(tpkey(owner), tpk)
}

// ReadStake returns the depositted stake and the token key that was committed to
func (kv *KV) ReadStake(owner PK) (amount uint64, tpk []byte) {
	k := skey(owner)
	v := kv.Get(k)
	if len(v) < 8 {
		return 0, nil
	}

	return binary.BigEndian.Uint64(v), kv.Get(tpkey(owner))
}

// CoinbaseTransfer is currency that is minted out of nothing and transferred to a receiver
func (kv *KV) CoinbaseTransfer(receiver PK, amount uint64) {
	k := append(receiver[:], []byte(balanceKey)...)
	v := kv.Get(k)
	if len(v) >= 8 {

		//amount is now the balance + the amount
		amount += binary.BigEndian.Uint64(v)
	} else {
		v = make([]byte, 8)
	}

	binary.BigEndian.PutUint64(v, amount)
	kv.Set(k, v)
}

// TransferCurrency will move currency from one account to the other
func (kv *KV) TransferCurrency(from, to PK, amount uint64) {
	if bytes.Equal(from[:], to[:]) {
		return
	}

	//read from balance
	balk := append(from[:], []byte(balanceKey)...)
	balv := kv.Get(balk)
	if len(balv) < 8 {
		return //no balance, do nothing
	}

	//check balance
	balance := binary.BigEndian.Uint64(balv)
	if balance < amount {
		return //nog enough balance don't do anything
	}

	kv.CoinbaseTransfer(to, amount)
	binary.BigEndian.PutUint64(balv, balance-amount)
	kv.Set(balk, balv)
}

// AccountBalance returns the current account balance of an identity
func (kv *KV) AccountBalance(pk PK) (b uint64) {
	k := append(pk[:], []byte(balanceKey)...)
	v := kv.Get(k)
	if len(v) < 8 {
		return 0
	}

	return binary.BigEndian.Uint64(v)
}

func skey(owner PK) []byte {
	return append(owner[:], []byte(stakeKey)...)
}
func tpkey(owner PK) []byte {
	return append(owner[:], []byte(tpkKey)...)
}
