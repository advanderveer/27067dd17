package tlog

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

// hashStorage provides an testing interface for storing hashes
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
	if fhs.f == nil {
		panic("failed to create temp file")
	}

	return
}

func (t *fileHashStorage) Rollback(n int64) {
	x := StoredHashIndex(0, n)
	err := t.f.Truncate(x * HashSize)
	if err != nil {
		panic(err)
	}

}

func (t *fileHashStorage) Append(hashes []Hash) hashStorage {
	for _, h := range hashes {
		_, err := t.f.Write(h[:])
		if err != nil {
			panic(err)
		}
	}

	return t
}

func (t *fileHashStorage) At(i int64) (h Hash) {
	_, err := t.f.ReadAt(h[:], i*HashSize)
	if err != nil {
		panic(err)
	}

	return
}

func (t *fileHashStorage) Len() (n int64) {
	off, _ := t.f.Seek(0, 1)
	n = off / HashSize
	return
}

func (t *fileHashStorage) ReadHashes(index []int64) (out []Hash, err error) {
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

// tileStorage is a testing interface that stores hash tiles
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
