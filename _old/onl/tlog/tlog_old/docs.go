package tlog

// import "crypto/sha256"
//
// // H is a cryptographic hash function is a deterministic function H that maps
// // an arbitrary-size message M to a small fixed-size output H(M), with the
// // property that it is infeasible in practice to produce any pair of distinct
// // messages M1 ≠ M2 with identical hashes H(M1) = H(M2). Of course, what is
// // feasible in practice changes over time.
// //
// // Hash functions allow bootstrapping a large amount of data by acting as a proof
// // that the data was not be tampered with.
// // But it can also act as a commitment that the hash can later be checked to see
// // if the person who send the hash did indeed send the correct bits
// func H(d []byte) (M [32]byte) {
// 	return sha256.Sum256(d)
// }
//
// // MerkleTree is constructed from N entries. N must be a power of 2.
// type MerkleTree struct {
// 	T [32]byte //Top level hash
// 	N uint64   //number of entries
// }
//
// // Coord describes a has by its coordinates in the tree
// type Coord struct {
// 	L uint64
// 	K uint64
// }
//
// // ProofExists to me that B is a record in the merkle tree. In general the proof
// // requires that a record is contained in the proof requires lg N hashes.
// func (tree *MerkleTree) ProofExists(B []byte) (proof map[Coord][32]byte) {
// 	return
// }
//
// // ProofPrefix returns a proof that the log of size N with tree hash T is a prefix
// // of larger log (N'> N) with three hash T'
// func (tree *MerkleTree) ProofPrefix(otherT [32]byte) {
//
// }
//
// // Store the tree on disk, requires only a few append-only files.
// func (tree *MerkleTree) Store() (err error) {
//
// 	// - first file holds log record data, concatenates.
// 	// - second is an index of the first, sequence of int allowing efficient access
// 	// to its record by its number.
// 	// - then a set of files for each level of the tree that is append only.
//
// 	// The file storage described earlier maintained lg N hash files, one for each level.
// 	// Using tiled storage, we only write the hash files for levels that are a multiple
// 	// of the tile height. For tiles of height 4, we’d only write the hash files
// 	// for levels 0, 4, 8, 12, 16, and so on. When we need a hash at another level,
// 	// we can read its tile and recompute the hash.
//
// 	return
// }
//
// // Each hash in the tree can be referred to by its coordinates L (level) and K (index).
// // We refer to it as h(L, K). So:
// //
// // h(0, K)	= H(record K)
// // h(L+1, K) = H( h(L, 2 K), h(L, 2 K+1) )
// func h(L, K uint64) (c Coord) {
// 	return Coord{L: L, K: K}
// }
//
// // TLog implements a transparent log. As described by: https://research.swtch.com/tlog
// // type TLog struct{}
// //
// // // Latest returns the current log size and top-level hash, cryptographically signed
// // // by the server so it can never deny its validity. Except by claiming to have
// // // been compromised entirely.
// // func (l *Tlog) Latest() {
// //
// // }
// //
// // // ProofRecord returns the proof that record R is contained in a tree of size N
// // func (l *Tlog) ProofRecord(R []byte, N uint64) {
// //
// // }
// //
// // // ProofTree returns the proof that the tree of size Na is a prefix of the tree
// // // of size Nb
// // func (l *Tlog) ProofTree(Na, Nb uint64) {
// //
// // }
// //
// // // ReadRecord reads a record
// // func (l *Tlog) ReadRecord(R []byte) {
// //
// // }
