package main

import (
	"math/big"
	"time"

	labs "github.com/FeurJak/Prudence/labs"

	precog "github.com/FeurJak/Prudence/pkg/precog"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
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
		NoP2P:        true, // disable P2P
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

	block := uint64(18958267)
	erc20Addr := common.HexToAddress("0x0f22e273Ed4b0E9289Bc3b663F1bc66e16C20238")

	// Setup local KV DB
	//NewPebbleDBDatabase(file string, cache int, handles int, namespace string, readonly, ephemeral bool) (ethdb.KeyValueStore, error)
	kvDb, err := NewPebbleDBDatabase("db", 512, utils.MakeDatabaseHandles(0), "lab", false, false)
	if err != nil {
		panic(err)
	}

	txEventTbl := NewTxEventTbl(kvDb, erc20Addr.Hex())
	lastEvent, err := txEventTbl.GetLatestEvent()
	if err != nil {
		panic(err)
	}

	if lastEvent != nil {
		labs.Logger.WithFields(lastEvent.LogFields()).Info("last transfer event")
		block = lastEvent.Block
	} else {
		labs.Logger.Info("no transfer events on record")
	}

	// Subscribe to new headers
	// get blocks and delete mempool Txs from it

	blocks := make(chan core.ChainHeadEvent)
	blockSub := psi.SubscribeChainHeadEvent(blocks)
	b, err := psi.GetLatestBlockNumber()
	if err != nil {
		panic("failed to get latest block number")
	}
	latestBlock := *b

	// iterate over blocks receipts until max block is reached & get stats of transfers
	labs.Logger.WithField("block", block).Info("starting from block")
	for {
		select {
		case <-blockSub.Err():
			panic("failed to subscribe to pending tx")
		case block := <-blocks:
			latestBlock = block.Block.NumberU64()
		default:
			if latestBlock >= block {
				receipts, _ := psi.GetReceiptsByNumber(block)
				if receipts == nil {
					panic("failed to get receipts")
				}
				for _, receipt := range receipts {
					for index, log := range receipt.Logs {
						if log.Address == erc20Addr {
							// Detected a transfer event
							if log.Topics[0].Hex() == "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef" {
								txEvent := NewTransferEvent(block, receipt.TxHash.Hex(), uint(index), common.BytesToAddress(log.Topics[1].Bytes()).Hex(), common.BytesToAddress(log.Topics[2].Bytes()).Hex(), new(big.Int).SetBytes(log.Data))
								index, err := txEventTbl.InsertEvent(txEvent, true)
								if err != nil {
									panic(err)
								}
								txEvent.Index = index
								labs.Logger.WithFields(txEvent.LogFields()).Info("inserted transfer event")

							}
						}
					}
				}
				block++
			}
		}
	}
	return nil
}
