// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tlog

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

type hashStorage interface {
	Len() int64
	At(i int64) Hash
	Append(hashes []Hash) hashStorage
	HashReader
}

type fileHashStorage struct {
	f *os.File
}

func newFileHashStorage() (fhs *fileHashStorage) {
	fhs = &fileHashStorage{}

	fhs.f, _ = ioutil.TempFile("", "")

	return
}

func (t fileHashStorage) Append(hashes []Hash) hashStorage {
	for _, h := range hashes {
		_, err := t.f.Write(h[:])
		if err != nil {
			panic(err)
		}
	}

	return t
}

func (t fileHashStorage) At(i int64) (h Hash) {
	_, err := t.f.ReadAt(h[:], i*HashSize)
	if err != nil {
		panic(err)
	}

	return
}

func (t fileHashStorage) Len() (n int64) {
	off, _ := t.f.Seek(0, 1)
	n = off / HashSize
	return
}

func (t fileHashStorage) ReadHashes(index []int64) (out []Hash, err error) {
	out = make([]Hash, len(index))
	for i, x := range index {
		var h Hash
		_, err = t.f.ReadAt(h[:], x*HashSize)
		if err != nil {
			panic(err)
		}

		out[i] = h
	}
	return
}

type testHashStorage []Hash

func (t testHashStorage) Append(hashes []Hash) hashStorage {
	return append(t, hashes...)
}

func (t testHashStorage) At(i int64) Hash {
	return t[i]
}

func (t testHashStorage) Len() int64 {
	return int64(len(t))
}

func (t testHashStorage) ReadHashes(index []int64) ([]Hash, error) {
	// It's not required by HashReader that indexes be in increasing order,
	// but check that the functions we are testing only ever ask for
	// indexes in increasing order.
	for i := 1; i < len(index); i++ {
		if index[i-1] >= index[i] {
			panic("indexes out of order")
		}
	}

	out := make([]Hash, len(index))
	for i, x := range index {
		out[i] = t[x]
	}
	return out, nil
}

type tileStorage interface {
	Unsaved() int
	TileReader
}

type dirTileStorage struct {
	height  int
	unsaved int
	dir     string
}

func newDirTileStorage(h int, dir string, tiles map[Tile][]byte) (dts *dirTileStorage) {
	dts = &dirTileStorage{height: h, dir: dir}

	for tile, data := range tiles {
		p := filepath.Join(dts.dir, tile.Path())
		fi, _ := os.Stat(p)
		if fi != nil {
			continue
		}

		err := os.MkdirAll(filepath.Dir(p), 0777)
		if err != nil {
			panic(err)
		}

		err = ioutil.WriteFile(p, data, 0777)
		if err != nil {
			panic(err)
		}
	}

	return dts
}

func (t dirTileStorage) Height() int {
	return t.height
}

func (t *dirTileStorage) SaveTiles(tiles []Tile, data [][]byte) {
	t.unsaved -= len(tiles)
	//@TODO this should be calle by the client to signal that a certain tile can
	// is valid and can be cached locally if necessary.
}

func (t dirTileStorage) Unsaved() int {
	return t.unsaved
}

func (t *dirTileStorage) ReadTiles(tiles []Tile) (out [][]byte, err error) {
	out = make([][]byte, len(tiles))
	for i, tile := range tiles {
		out[i], err = ioutil.ReadFile(filepath.Join(t.dir, tile.Path()))
		if err != nil {
			panic(err)
		}
	}

	t.unsaved += len(tiles)
	return out, nil
}

type testTilesStorage struct {
	unsaved int
	m       map[Tile][]byte
	height  int
}

func (t testTilesStorage) Height() int {
	return t.height
}

func (t *testTilesStorage) SaveTiles(tiles []Tile, data [][]byte) {
	t.unsaved -= len(tiles)
}

func (t *testTilesStorage) ReadTiles(tiles []Tile) ([][]byte, error) {
	out := make([][]byte, len(tiles))
	for i, tile := range tiles {
		out[i] = t.m[tile]
	}
	t.unsaved += len(tiles)
	return out, nil
}

func (t testTilesStorage) Unsaved() int {
	return t.unsaved
}

func TestTree(t *testing.T) {
	const testH = 4 //tile height

	dir, _ := ioutil.TempDir("", "")
	if dir == "" {
		panic("failed to open temp dir")
	}

	for _, c := range []struct {
		hs hashStorage
		tr func(tiles map[Tile][]byte) tileStorage
	}{
		{hs: testHashStorage{}, tr: func(tiles map[Tile][]byte) tileStorage { return &testTilesStorage{height: testH, m: tiles} }}, //an in-memory test storage for hashes
		{hs: newFileHashStorage(), tr: func(tiles map[Tile][]byte) tileStorage { return newDirTileStorage(testH, dir, tiles) }},    //persistent append-only file
	} {
		t.Run(fmt.Sprintf("%T", c.hs), func(t *testing.T) {
			storage := c.hs

			var trees []Hash
			var leafhashes []Hash
			tiles := make(map[Tile][]byte)

			for i := int64(0); i < 40; i++ { //was originally 100
				data := []byte(fmt.Sprintf("leaf %d", i))

				hashes, err := StoredHashes(i, data, storage)
				if err != nil {
					t.Fatal(err)
				}

				leafhashes = append(leafhashes, RecordHash(data))
				oldStorage := storage.Len()
				storage = storage.Append(hashes)

				if count := StoredHashCount(i + 1); count != storage.Len() {
					t.Errorf("StoredHashCount(%d) = %d, have %d StoredHashes", i+1, count, storage.Len())
				}

				th, err := TreeHash(i+1, storage)
				if err != nil {
					t.Fatal(err)
				}

				for _, tile := range NewTiles(testH, i, i+1) {
					data, err := ReadTileData(tile, storage)
					if err != nil {
						t.Fatal(err)
					}
					old := Tile{H: tile.H, L: tile.L, N: tile.N, W: tile.W - 1}
					oldData := tiles[old]
					if len(oldData) != len(data)-HashSize || !bytes.Equal(oldData, data[:len(oldData)]) {
						t.Fatalf("tile %v not extending earlier tile %v", tile.Path(), old.Path())
					}
					tiles[tile] = data
				}
				for _, tile := range NewTiles(testH, 0, i+1) {
					data, err := ReadTileData(tile, storage)
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(tiles[tile], data) {
						t.Fatalf("mismatch at %+v", tile)
					}
				}
				for _, tile := range NewTiles(testH, i/2, i+1) {
					data, err := ReadTileData(tile, storage)
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(tiles[tile], data) {
						t.Fatalf("mismatch at %+v", tile)
					}
				}

				// Check that all the new hashes are readable from their tiles.
				for j := oldStorage; j < storage.Len(); j++ {
					tile := TileForIndex(testH, int64(j))
					data, ok := tiles[tile]
					if !ok {
						t.Log(NewTiles(testH, 0, i+1))
						t.Fatalf("TileForIndex(%d, %d) = %v, not yet stored (i=%d, stored %d)", testH, j, tile.Path(), i, storage.Len())
						continue
					}

					h, err := HashFromTile(tile, data, int64(j))
					if err != nil {
						t.Fatal(err)
					}
					if h != storage.At(j) {
						t.Errorf("HashFromTile(%v, %d) = %v, want %v", tile.Path(), int64(j), h, storage.At(j))
					}
				}

				trees = append(trees, th)

				// Check that leaf proofs work, for all trees and leaves so far.
				for j := int64(0); j <= i; j++ {
					p, err := ProveRecord(i+1, j, storage)
					if err != nil {
						t.Fatalf("ProveRecord(%d, %d): %v", i+1, j, err)
					}
					if err := CheckRecord(p, i+1, th, j, leafhashes[j]); err != nil {
						t.Fatalf("CheckRecord(%d, %d): %v", i+1, j, err)
					}
					for k := range p {
						p[k][0] ^= 1
						if err := CheckRecord(p, i+1, th, j, leafhashes[j]); err == nil {
							t.Fatalf("CheckRecord(%d, %d) succeeded with corrupt proof hash #%d!", i+1, j, k)
						}
						p[k][0] ^= 1
					}
				}

				// Check that leaf proofs work using TileReader.
				// To prove a leaf that way, all you have to do is read and verify its hash.
				storage := c.tr(tiles)

				thr := TileHashReader(Tree{i + 1, th}, storage)
				for j := int64(0); j <= i; j++ {
					h, err := thr.ReadHashes([]int64{StoredHashIndex(0, j)})
					if err != nil {
						t.Fatalf("TileHashReader(%d).ReadHashes(%d): %v", i+1, j, err)
					}
					if h[0] != leafhashes[j] {
						t.Fatalf("TileHashReader(%d).ReadHashes(%d) returned wrong hash", i+1, j)
					}

					// Even though reading the hash suffices,
					// check we can generate the proof too.
					p, err := ProveRecord(i+1, j, thr)
					if err != nil {
						t.Fatalf("ProveRecord(%d, %d, TileHashReader(%d)): %v", i+1, j, i+1, err)
					}
					if err := CheckRecord(p, i+1, th, j, leafhashes[j]); err != nil {
						t.Fatalf("CheckRecord(%d, %d, TileHashReader(%d)): %v", i+1, j, i+1, err)
					}
				}
				if storage.Unsaved() != 0 {
					t.Fatalf("TileHashReader(%d) did not save %d tiles", i+1, storage.Unsaved())
				}

				// Check that ReadHashes will give an error if the index is not in the tree.
				if _, err := thr.ReadHashes([]int64{(i + 1) * 2}); err == nil {
					t.Fatalf("TileHashReader(%d).ReadHashes(%d) for index not in tree <nil>, want err", i, i+1)
				}
				if storage.Unsaved() != 0 {
					t.Fatalf("TileHashReader(%d) did not save %d tiles", i+1, storage.Unsaved())
				}

				// Check that tree proofs work, for all trees so far, using TileReader.
				// To prove a tree that way, all you have to do is compute and verify its hash.
				for j := int64(0); j <= i; j++ {
					h, err := TreeHash(j+1, thr)
					if err != nil {
						t.Fatalf("TreeHash(%d, TileHashReader(%d)): %v", j, i+1, err)
					}
					if h != trees[j] {
						t.Fatalf("TreeHash(%d, TileHashReader(%d)) = %x, want %x (%v)", j, i+1, h[:], trees[j][:], trees[j])
					}

					// Even though computing the subtree hash suffices,
					// check that we can generate the proof too.
					p, err := ProveTree(i+1, j+1, thr)
					if err != nil {
						t.Fatalf("ProveTree(%d, %d): %v", i+1, j+1, err)
					}
					if err := CheckTree(p, i+1, th, j+1, trees[j]); err != nil {
						t.Fatalf("CheckTree(%d, %d): %v [%v]", i+1, j+1, err, p)
					}
					for k := range p {
						p[k][0] ^= 1
						if err := CheckTree(p, i+1, th, j+1, trees[j]); err == nil {
							t.Fatalf("CheckTree(%d, %d) succeeded with corrupt proof hash #%d!", i+1, j+1, k)
						}
						p[k][0] ^= 1
					}
				}
				if storage.Unsaved() != 0 {
					t.Fatalf("TileHashReader(%d) did not save %d tiles", i+1, storage.Unsaved())
				}
			}

		})
	}

}

func TestSplitStoredHashIndex(t *testing.T) {
	for l := 0; l < 10; l++ {
		for n := int64(0); n < 100; n++ {
			x := StoredHashIndex(l, n)
			l1, n1 := SplitStoredHashIndex(x)
			if l1 != l || n1 != n {
				t.Fatalf("StoredHashIndex(%d, %d) = %d, but SplitStoredHashIndex(%d) = %d, %d", l, n, x, x, l1, n1)
			}
		}
	}
}

// TODO(rsc): Test invalid paths too, like "tile/3/5/123/456/078".
var tilePaths = []struct {
	path string
	tile Tile
}{
	{"tile/4/0/001", Tile{4, 0, 1, 16}},
	{"tile/4/0/001.p/5", Tile{4, 0, 1, 5}},
	{"tile/3/5/x123/x456/078", Tile{3, 5, 123456078, 8}},
	{"tile/3/5/x123/x456/078.p/2", Tile{3, 5, 123456078, 2}},
	{"tile/1/0/x003/x057/500", Tile{1, 0, 3057500, 2}},
	{"tile/3/5/123/456/078", Tile{}},
	{"tile/3/-1/123/456/078", Tile{}},
	{"tile/1/data/x003/x057/500", Tile{1, -1, 3057500, 2}},
}

func TestTilePath(t *testing.T) {
	for _, tt := range tilePaths {
		if tt.tile.H > 0 {
			p := tt.tile.Path()
			if p != tt.path {
				t.Errorf("%+v.Path() = %q, want %q", tt.tile, p, tt.path)
			}
		}
		tile, err := ParseTilePath(tt.path)
		if err != nil {
			if tt.tile.H == 0 {
				// Expected error.
				continue
			}
			t.Errorf("ParseTilePath(%q): %v", tt.path, err)
		} else if tile != tt.tile {
			if tt.tile.H == 0 {
				t.Errorf("ParseTilePath(%q): expected error, got %+v", tt.path, tt.tile)
				continue
			}
			t.Errorf("ParseTilePath(%q) = %+v, want %+v", tt.path, tile, tt.tile)
		}
	}
}
