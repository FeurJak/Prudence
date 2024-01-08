package main

import (
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/pebble"
)

const (
	txEventTbl_prefix = "tx_event"
	accTbl_prefix     = "account"
)

func NewPebbleDBDatabase(file string, cache int, handles int, namespace string, readonly, ephemeral bool) (ethdb.KeyValueStore, error) {
	db, err := pebble.New(file, cache, handles, namespace, readonly, ephemeral)
	if err != nil {
		return nil, err
	}
	return db, nil
}
