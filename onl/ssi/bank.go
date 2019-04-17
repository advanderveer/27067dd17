package ssi

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
)

var (
	//ErrAccountExists is returned when it is expected the account doesn't exist
	ErrAccountExists = errors.New("account already exist")

	//ErrAccountNotExist is returend when the account is expected to exist
	ErrAccountNotExist = errors.New("account doesn't exist")

	//ErrNotEnoughFunds is returned when an account doesn't have enough funds
	ErrNotEnoughFunds = errors.New("not enough funds")
)

//Bank manages the funds of accounts
type Bank struct {
	db *DB
}

//NewBank sets up a new bank
func NewBank() *Bank {
	return &Bank{db: NewDB()}
}

//OpenAccount creates a new account, it should not exist
func (b *Bank) OpenAccount(name string, balance int64) (err error) {
	tx := b.db.NewTx()
	if tx.Get([]byte(name)) != nil {
		return ErrAccountExists
	}

	tx.Set([]byte(name), int64b(balance))
	return tx.Commit()
}

//TransferFunds transfers funds between accounts
func (b *Bank) TransferFunds(amount int64, from, to string) (err error) {
	tx := b.db.NewTx()
	fromr := tx.Get([]byte(from))
	tor := tx.Get([]byte(to))
	if fromr == nil || tor == nil {
		return ErrAccountNotExist
	}

	if from == to {
		return nil //no-op
	}

	fromb := bint64(fromr)
	if fromb < amount {
		return ErrNotEnoughFunds
	}

	tob := bint64(tor)
	tx.Set([]byte(to), int64b(tob+amount))
	tx.Set([]byte(from), int64b(fromb-amount))

	//the logic below is purley to test if anything gets lost if transactions
	//would be send over the network before being committed somewhere else
	buf := bytes.NewBuffer(nil)
	err = gob.NewEncoder(buf).Encode(tx.data)
	if err != nil {
		return fmt.Errorf("failed to encode: %v", err)
	}

	txd := &TxData{}
	err = gob.NewDecoder(buf).Decode(txd)
	if err != nil {
		return fmt.Errorf("failed to decode: %v", err)
	}

	return b.db.Commit(txd)
}

//CurrentBalance returns the current  balance of an account
func (b *Bank) CurrentBalance(name string) (balance int64, err error) {
	tx := b.db.NewTx()
	raw := tx.Get([]byte(name))
	if raw == nil {
		return 0, ErrAccountNotExist
	}

	return bint64(raw), tx.Commit()
}

func int64b(n int64) (b []byte) {
	buf := bytes.NewBuffer(nil)
	gob.NewEncoder(buf).Encode(n)
	return buf.Bytes()
}

func bint64(b []byte) (n int64) {
	r := bytes.NewReader(b)
	gob.NewDecoder(r).Decode(&n)
	return
}
