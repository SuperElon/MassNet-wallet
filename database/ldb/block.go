// Modified for MassNet
// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ldb

import (
	"bytes"
	"encoding/binary"

	"github.com/massnetorg/MassNet-wallet/logging"
	wirepb "github.com/massnetorg/MassNet-wallet/wire/pb"

	"github.com/golang/protobuf/proto"

	"github.com/massnetorg/MassNet-wallet/database"
	"github.com/massnetorg/MassNet-wallet/massutil"
	"github.com/massnetorg/MassNet-wallet/wire"

	"github.com/btcsuite/goleveldb/leveldb"
)

// FetchBlockBySha - return a massutil Block
func (db *LevelDb) FetchBlockBySha(sha *wire.Hash) (blk *massutil.Block, err error) {
	return db.fetchBlockBySha(sha)
}

// fetchBlockBySha - return a massutil Block
// Must be called with db lock held.
func (db *LevelDb) fetchBlockBySha(sha *wire.Hash) (blk *massutil.Block, err error) {
	buf, height, err := db.fetchSha(sha)
	if err != nil {
		return
	}

	blk, err = massutil.NewBlockFromBytes(buf, wire.DB)
	if err != nil {
		return
	}
	blk.SetHeight(height)

	return
}

// FetchBlockHeightBySha returns the block height for the given hash.  This is
// part of the database.Db interface implementation.
func (db *LevelDb) FetchBlockHeightBySha(sha *wire.Hash) (int32, error) {
	return db.getBlkLoc(sha)
}

// FetchBlockHeaderBySha - return a Hash
func (db *LevelDb) FetchBlockHeaderBySha(sha *wire.Hash) (bh *wire.BlockHeader, err error) {

	buf, _, err := db.fetchSha(sha)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(buf)

	blockBaseLength, err := wire.ReadVarInt(r, 0)
	if err != nil {
		return nil, err
	}

	baseData := make([]byte, blockBaseLength, blockBaseLength)
	_, err = r.Read(baseData)
	if err != nil {
		return nil, err
	}
	basePb := new(wirepb.BlockBase)
	err = proto.Unmarshal(baseData, basePb)
	if err != nil {
		return nil, err
	}
	base, err := wire.NewBlockBaseFromProto(basePb)
	if err != nil {
		return nil, err
	}

	bh = &base.Header

	return bh, nil
}

func (db *LevelDb) getBlkLoc(sha *wire.Hash) (int32, error) {
	key := shaBlkToKey(sha)

	data, err := db.lDb.Get(key, db.ro)
	if err != nil {
		if err == leveldb.ErrNotFound {
			err = database.ErrBlockShaMissing
		}
		return 0, err
	}

	blkHeight := binary.LittleEndian.Uint64(data)

	return int32(blkHeight), nil
}

func (db *LevelDb) getBlkByHeight(blkHeight int32) (rsha *wire.Hash, rbuf []byte, err error) {
	var blkVal []byte

	key := int64ToKey(int64(blkHeight))

	blkVal, err = db.lDb.Get(key, db.ro)
	if err != nil {
		logging.CPrint(logging.TRACE, "failed to find block on height", logging.LogFormat{"height": blkHeight})
		return
	}

	var sha wire.Hash

	sha.SetBytes(blkVal[0:32])

	blockdata := make([]byte, len(blkVal[32:]))
	copy(blockdata[:], blkVal[32:])

	return &sha, blockdata, nil
}

func (db *LevelDb) getBlk(sha *wire.Hash) (rblkHeight int32, rbuf []byte, err error) {
	var blkHeight int32

	blkHeight, err = db.getBlkLoc(sha)
	if err != nil {
		return
	}

	var buf []byte

	_, buf, err = db.getBlkByHeight(blkHeight)
	if err != nil {
		return
	}
	return blkHeight, buf, nil
}

func (db *LevelDb) setBlk(sha *wire.Hash, blkHeight int32, buf []byte) {
	var lw [8]byte
	binary.LittleEndian.PutUint64(lw[0:8], uint64(blkHeight))

	shaKey := shaBlkToKey(sha)
	blkKey := int64ToKey(int64(blkHeight))

	blkVal := make([]byte, len(sha)+len(buf))
	copy(blkVal[0:], sha[:])
	copy(blkVal[len(sha):], buf)

	db.lBatch().Put(shaKey, lw[:])
	db.lBatch().Put(blkKey, blkVal)
}

// insertSha stores a block hash and its associated data block with a
// previous sha of `prevSha'.
// insertSha shall be called with db lock held
func (db *LevelDb) insertBlockData(sha *wire.Hash, prevSha *wire.Hash, buf []byte) (int32, error) {
	oBlkHeight, err := db.getBlkLoc(prevSha)
	if err != nil {
		oBlkHeight = -1
		if db.nextBlock != 0 {
			return 0, err
		}
	}

	blkHeight := oBlkHeight + 1

	db.setBlk(sha, blkHeight, buf)

	db.lastBlkShaCached = true
	db.lastBlkSha = *sha
	db.lastBlkIdx = blkHeight
	db.nextBlock = blkHeight + 1

	return blkHeight, nil
}

// fetchSha returns the datablock for the given Hash.
func (db *LevelDb) fetchSha(sha *wire.Hash) (rbuf []byte,
	rblkHeight int32, err error) {
	var blkHeight int32
	var buf []byte

	blkHeight, buf, err = db.getBlk(sha)
	if err != nil {
		return
	}

	return buf, blkHeight, nil
}

// ExistsSha looks up the given block hash
// returns true if it is present in the database.
func (db *LevelDb) ExistsSha(sha *wire.Hash) (bool, error) {
	return db.blkExistsSha(sha)
}

// blkExistsSha looks up the given block hash
// returns true if it is present in the database.
// CALLED WITH LOCK HELD
func (db *LevelDb) blkExistsSha(sha *wire.Hash) (bool, error) {
	key := shaBlkToKey(sha)

	return db.lDb.Has(key, db.ro)
}

// FetchBlockShaByHeight returns a block hash based on its height in the
// block chain.
func (db *LevelDb) FetchBlockShaByHeight(height int32) (sha *wire.Hash, err error) {

	return db.fetchBlockShaByHeight(height)
}

// fetchBlockShaByHeight returns a block hash based on its height in the
// block chain.
func (db *LevelDb) fetchBlockShaByHeight(height int32) (rsha *wire.Hash, err error) {
	key := int64ToKey(int64(height))

	blkVal, err := db.lDb.Get(key, db.ro)
	if err != nil {
		logging.CPrint(logging.TRACE, "failed to find block on height", logging.LogFormat{"height": height})
		return
	}

	var sha wire.Hash
	sha.SetBytes(blkVal[0:32])

	return &sha, nil
}

// FetchHeightRange looks up a range of blocks by the start and ending
// heights.  Fetch is inclusive of the start height and exclusive of the
// ending height. To fetch all hashes from the start height until no
// more are present, use the special id `AllShas'.
func (db *LevelDb) FetchHeightRange(startHeight, endHeight int32) (rshalist []wire.Hash, err error) {
	shalist := make([]wire.Hash, 0, endHeight-startHeight)
	for height := startHeight; height < endHeight; height++ {
		key := int64ToKey(int64(height))
		blkVal, lerr := db.lDb.Get(key, db.ro)
		if lerr != nil {
			break
		}

		var sha wire.Hash
		sha.SetBytes(blkVal[0:32])
		shalist = append(shalist, sha)
	}

	if err != nil {
		return
	}

	return shalist, nil
}

// NewestSha returns the hash and block height of the most recent (end) block of
// the block chain.  It will return the zero hash, -1 for the block height, and
// no error (nil) if there are not any blocks in the database yet.
func (db *LevelDb) NewestSha() (rsha *wire.Hash, rblkid int32, err error) {

	if db.lastBlkIdx == -1 {
		return &wire.Hash{}, -1, nil
	}
	sha := db.lastBlkSha

	return &sha, db.lastBlkIdx, nil
}

// checkAddrIndexVersion returns an error if the address index version stored
// in the database is less than the current version, or if it doesn't exist.
// This function is used on startup to signal OpenDB to drop the address index
// if it's in an old, incompatible format.
func (db *LevelDb) checkAddrIndexVersion() error {

	data, err := db.lDb.Get(addrIndexVersionKey, db.ro)
	if err != nil {
		return database.ErrAddrIndexDoesNotExist
	}

	indexVersion := binary.LittleEndian.Uint16(data)

	if indexVersion != uint16(addrIndexCurrentVersion) {
		return database.ErrAddrIndexDoesNotExist
	}

	return nil
}

// fetchAddrIndexTip returns the last block height and block sha to be indexed.
// Meta-data about the address tip is currently cached in memory, and will be
// updated accordingly by functions that modify the state. This function is
// used on start up to load the info into memory. Callers will use the public
// version of this function below, which returns our cached copy.
func (db *LevelDb) fetchAddrIndexTip() (*wire.Hash, int32, error) {

	data, err := db.lDb.Get(addrIndexMetaDataKey, db.ro)
	if err != nil {
		return &wire.Hash{}, -1, database.ErrAddrIndexDoesNotExist
	}

	var blkSha wire.Hash
	blkSha.SetBytes(data[0:32])

	blkHeight := binary.LittleEndian.Uint64(data[32:])

	return &blkSha, int32(blkHeight), nil
}

// FetchAddrIndexTip returns the hash and block height of the most recent
// block whose transactions have been indexed by address. It will return
// ErrAddrIndexDoesNotExist along with a zero hash, and -1 if the
// addrindex hasn't yet been built up.
func (db *LevelDb) FetchAddrIndexTip() (*wire.Hash, int32, error) {

	if db.lastAddrIndexBlkIdx == -1 {
		return &wire.Hash{}, -1, database.ErrAddrIndexDoesNotExist
	}
	sha := db.lastAddrIndexBlkSha

	return &sha, db.lastAddrIndexBlkIdx, nil
}
