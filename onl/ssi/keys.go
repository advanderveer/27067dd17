package ssi

import (
	"fmt"
	"hash/fnv"
)

//Change represents a key write
type Change struct {
	K []byte
	V []byte
}

//KeyChangeSet is a set of transaction keys with their values
type KeyChangeSet map[uint64]*Change

//KeySet returns just the set of keys without any values
func (kvs KeyChangeSet) KeySet() (ks KeySet) {
	ks = make(KeySet, len(kvs))
	for k := range kvs {
		ks[k] = struct{}{}
	}

	return
}

//Add a key to our set
func (kvs KeyChangeSet) Add(k, v []byte) {
	kvs[keyHash(k)] = &Change{K: k, V: v}
}

//KeySet is a set of transaction keys
type KeySet map[uint64]struct{}

//Add a key to our set
func (ks KeySet) Add(k []byte) {
	ks[keyHash(k)] = struct{}{}
}

func keyHash(k []byte) uint64 {
	//@TODO document what effect do collisions have? Logic sugges that the hash
	//then covers (in effect) more then one actual key, this means that any reads
	//would read extra values which can cause extra conflicts. I believe it shouln't
	//cause less conflicts (i.e corrupted database). Badger authors also came to
	//this conclusion: https://blog.dgraph.io/post/badger-txn/

	h := fnv.New64a()
	n, err := h.Write(k)
	if err != nil || n != len(k) {
		panic(fmt.Sprintf("failed to hash write key (n: %d, len: %d): %v", n, len(k), err))
	}

	return h.Sum64()
}
