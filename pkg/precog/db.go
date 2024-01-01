package precog

import (
	"fmt"
	"time"

	pDb "github.com/Prudence/pkg/precog/core/db"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (b *Backend) initStateDb() (err error) {
	var (
		db      ethdb.Database
		stateDb state.Database
	)
	if b.conf.State.Remote {
		// remote connection to state maintained by a geth node
		// this requires archive node to access latest states  (run your node with --gcmode=archive)
		//b.db = pDb.NewAPI(pDb.GetRemoteEthDb(b.gethCli.RpcCl()))
		db = pDb.GetRemoteEthDb(b.gethCli.RpcCl())
	} else {
		// access state from disk
		// disk is locked by geth node so make sure to switch that off first
		db, err = pDb.GetDiskEthDb(b.conf.State.DbOpts)
		if err != nil {
			return errors.Wrap(err, "failed to get disk ethdb")
		}
	}

	stateDb = pDb.GetStateDb(b.conf.State.TrieCacheConf, db)

	b.db = pDb.NewAPI(db, stateDb)

	block, err := b.db.GetLatestBlock()
	if err != nil {
		return fmt.Errorf("failed to get latest block: %s", err)
	}

	b.lastHead = block.Header()
	b.logger.WithFields(logrus.Fields{
		"lastHead":   b.lastHead.Hash(),
		"lastNumber": block.Number(),
		"headRoot":   b.lastHead.Root,
	}).Info("loaded DB")

	return b.verifyStateDb()
}

func (b *Backend) verifyStateDb() (err error) {
	// Check state loading
	stateLoad := false
	testRoot := b.lastHead.Root
	for i := 0; i < 10; i++ {
		latestState, err := b.StateAt(testRoot)
		if err != nil {
			b.logger.WithFields(logrus.Fields{
				"stateRoot": testRoot,
				"err":       err,
			}).Warn("unable to get state")
			time.Sleep(15 * time.Second)
			continue
		}
		// get nonce of random address
		nonce := latestState.GetNonce(common.HexToAddress("0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5"))
		b.logger.WithFields(logrus.Fields{
			"nonce": nonce,
			"state": testRoot,
		}).Info("got nonce of random addr")
		stateLoad = true
		break
	}
	if !stateLoad {
		return fmt.Errorf("failed to a recent state: %s", err)
	}
	return nil
}

// StateAt returns a state database for a given root hash (generally the head).
func (b *Backend) StateAt(root common.Hash) (*state.StateDB, error) {
	return b.db.StateAt(root)
}

// GetBlock retrieves a specific block from state
func (b *Backend) GetBlock(hash common.Hash, number uint64) *types.Block {
	block, err := b.db.GetBlock(hash, number)
	if err != nil {
		b.logger.WithFields(logrus.Fields{
			"err": err,
		}).Error("failed to get block")
		return nil
	}
	return block
}

// GetHeader retrieves a specific header from state
func (b *Backend) GetHeader(hash common.Hash, number uint64) *types.Header {
	block, err := b.db.GetBlock(hash, number)
	if err != nil {
		b.logger.WithFields(logrus.Fields{
			"err": err,
		}).Error("failed to get header")
		return nil
	}
	return block.Header()
}
