package db

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/remotedb"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/triedb/hashdb"
)

func GetRemoteEthDb(client *rpc.Client) ethdb.Database {
	return remotedb.New(client) // Connect to Existing DB remotely
}

func GetDiskEthDb(o rawdb.OpenOptions) (ethDb ethdb.Database, err error) {
	ethDb, err = rawdb.Open(o)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %s", err)
	}
	return ethDb, nil
}

func GetStateDb(trieCacheConf *core.CacheConfig, db ethdb.Database) (stateDb state.Database) {
	// Open trie database with provided config
	trieConf := &trie.Config{Preimages: trieCacheConf.Preimages}
	// create trie.Config
	trieConf.HashDB = &hashdb.Config{
		CleanCacheSize: trieCacheConf.TrieCleanLimit * 1024 * 1024,
	}
	return state.NewDatabaseWithNodeDB(db, trie.NewDatabase(db, trieConf))
}
