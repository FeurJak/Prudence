package main

import (
	"sync"
	"time"

	labs "github.com/Prudence/labs"
	precog "github.com/Prudence/pkg/precog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var runCommand = &cli.Command{
	Action:    runCmd,
	Name:      "run",
	Usage:     "run lab",
	ArgsUsage: "<connection string>",
}

func runCmd(ctx *cli.Context) error {

	connStr := ctx.Args().First()

	psi, err := precog.NewBackend(labs.Logger, &precog.BackendConfig{
		GethConnAddr: connStr,
		State: precog.StateConf{
			Remote: true,
			// default values used in core / blockchain.go defaultCacheConfig
			TrieCacheConf: &core.CacheConfig{
				TrieCleanLimit: 256,
				TrieDirtyLimit: 256,
				TrieTimeLimit:  5 * time.Minute,
				SnapshotLimit:  256,
				SnapshotWait:   false,
				Preimages:      true,
			},
		},
	})
	if err != nil {
		panic("failed to init backend, err: " + err.Error())
	}

	mempool := make(map[common.Hash]bool)
	mempoolMtx := new(sync.RWMutex)

	go func() {
		pendingTxChan := make(chan core.NewTxsEvent)
		pendingSub := psi.SubToPendingTx(pendingTxChan)
		for {
			select {
			case <-pendingSub.Err():
				panic("failed to subscribe to pending tx")
			case txs := <-pendingTxChan:
				for _, tx := range txs.Txs {
					if _, ok := mempool[tx.Hash()]; !ok {
						mempoolMtx.Lock()
						mempool[tx.Hash()] = true
						mempoolMtx.Unlock()
					}
				}
			}
		}
	}()

	// get blocks and delete mempool Txs from it
	blocks := make(chan core.ChainHeadEvent)
	blockSub := psi.SubscribeChainHeadEvent(blocks)
	for {
		select {
		case <-blockSub.Err():
			panic("failed to subscribe to pending tx")
		case block := <-blocks:
			known := 0
			txs := block.Block.Transactions()
			for _, tx := range txs {
				if _, ok := mempool[tx.Hash()]; ok {
					mempoolMtx.Lock()
					delete(mempool, tx.Hash())
					mempoolMtx.Unlock()
					known++
				}
			}
			labs.Logger.WithFields(logrus.Fields{
				"Txs":         len(txs),
				"Pending Txs": len(mempool),
				"Known":       known,
				"Unkown":      len(txs) - known,
				"Coverage":    float64(known) / float64(len(txs)),
			}).Info("received new block")
		}
	}
}
