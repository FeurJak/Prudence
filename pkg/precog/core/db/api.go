package db

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
)

type API struct {
	db      ethdb.Database
	stateDb state.Database
}

func NewAPI(db ethdb.Database, stateDb state.Database) *API {
	return &API{db: db, stateDb: stateDb}
}

func (a *API) GetHead() (head common.Hash, err error) {
	head = rawdb.ReadHeadBlockHash(a.db)
	if head == (common.Hash{}) {
		return common.Hash{}, fmt.Errorf("empty db found, can't read latest head hash")
	}
	return head, nil
}

func (a *API) GetHeaderNumber(hash common.Hash) (number *uint64, err error) {
	number = rawdb.ReadHeaderNumber(a.db, hash)
	if number == nil {
		return nil, fmt.Errorf("unable to read header number for hash")
	}
	return number, nil
}

func (a *API) GetBlock(hash common.Hash, number uint64) (block *types.Block, err error) {
	block = rawdb.ReadBlock(a.db, hash, number)
	if block == nil {
		return nil, fmt.Errorf("unable to get block from db")
	}
	return block, nil
}

func (a *API) GetLatestBlock() (block *types.Block, err error) {
	head, err := a.GetHead()
	if err != nil {
		return nil, err
	}
	number, err := a.GetHeaderNumber(head)
	if err != nil {
		return nil, err
	}
	block, err = a.GetBlock(head, *number)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (a *API) StateAt(root common.Hash) (*state.StateDB, error) {
	return state.New(root, a.stateDb, nil)
}

func (a *API) EthDB() ethdb.Database {
	return a.db
}
