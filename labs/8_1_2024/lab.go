package main

import (
	"math/big"
	"time"

	labs "github.com/FeurJak/Prudence/labs"
	"github.com/pterm/pterm"
	"github.com/sirupsen/logrus"

	precog "github.com/FeurJak/Prudence/pkg/precog"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/urfave/cli/v2"
)

var runCommand = &cli.Command{
	Action:    runCmd,
	Name:      "run",
	Usage:     "run lab",
	ArgsUsage: "<connection string>",
}

var accCommand = &cli.Command{
	Action:    accCmd,
	Name:      "accounts",
	Usage:     "accounts",
	ArgsUsage: "<connection string>",
}

type LabEnv struct {
	psi         precog.API
	kvDb        ethdb.KeyValueStore
	txEventTbl  TxEventTbl
	accRegistry *AccRegistry
}

func (env *LabEnv) InitLabEnv(connStr string, erc20Addr string) (err error) {
	env.psi, err = precog.NewBackend(labs.Logger, &precog.BackendConfig{
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

	// Setup local KV DB
	env.kvDb, err = NewPebbleDBDatabase("db", 512, utils.MakeDatabaseHandles(0), "lab", false, false)
	if err != nil {
		panic(err)
	}

	env.txEventTbl = NewTxEventTbl(env.kvDb, erc20Addr)
	env.accRegistry = NewAccRegistry(env.kvDb, erc20Addr)

	return nil
}

func accCmd(ctx *cli.Context) error {
	erc20Addr := common.HexToAddress("0x6349ae2681f495fe1d737e49b8f402e4577e9b4d")

	env := new(LabEnv)
	err := env.InitLabEnv(ctx.Args().First(), erc20Addr.Hex())
	if err != nil {
		panic("failed to init env, err: " + err.Error())
	}

	accs, err := env.accRegistry.LatestAccounts()

	tableData := pterm.TableData{}
	tableData = append(tableData, accs[0].GetFields())

	for _, acc := range accs {
		tableData = append(tableData, acc.Serialize())
	}

	pterm.DefaultTable.WithHasHeader().WithBoxed().WithData(tableData).Render()

	return nil
}

func runCmd(ctx *cli.Context) error {

	block := uint64(18952399)
	erc20Addr := common.HexToAddress("0x6349ae2681f495fe1d737e49b8f402e4577e9b4d")
	env := new(LabEnv)
	err := env.InitLabEnv(ctx.Args().First(), erc20Addr.Hex())
	if err != nil {
		panic("failed to init env, err: " + err.Error())
	}

	/*
		multi := pterm.DefaultMultiPrinter
		pb1, _ := pterm.DefaultProgressbar.WithTotal(1000000).WithWriter(multi.NewWriter()).Start("Progress")
		pb1.ShowPercentage = true
	*/

	lastEvent, err := env.txEventTbl.GetLatestEvent()
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
	blockSub := env.psi.SubscribeChainHeadEvent(blocks)
	b, err := env.psi.GetLatestBlockNumber()
	if err != nil {
		panic("failed to get latest block number")
	}
	latestBlock := *b

	// iterate over blocks receipts until max block is reached & get stats of transfers
	labs.Logger.WithField("block", block).Info("starting from block")
	//multi.Start()
	for {
		select {
		case <-blockSub.Err():
			panic("failed to subscribe to pending tx")
		case newBlock := <-blocks:

			latestBlock = newBlock.Block.NumberU64()
			prog := float64(block) / float64(latestBlock) * 100

			total, err := env.accRegistry.TotalAddress()
			if err != nil {
				panic(err)
			}
			labs.Logger.WithFields(
				logrus.Fields{
					"prog":      prog,
					"lastBlock": latestBlock,
					"currBlock": block,
					"total":     total,
				},
			).Info("total addresses found")

			//
			//pb1.Current = prog
			//pb1.Increment()

		default:
			if latestBlock >= block {
				receipts, _ := env.psi.GetReceiptsByNumber(block)
				if receipts == nil {
					panic("failed to get receipts")
				}
				for _, receipt := range receipts {
					for index, log := range receipt.Logs {
						if log.Address == erc20Addr {
							// Detected a transfer event
							if log.Topics[0].Hex() == "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef" {
								txEvent := NewTransferEvent(block, receipt.TxHash.Hex(), uint(index), common.BytesToAddress(log.Topics[1].Bytes()).Hex(), common.BytesToAddress(log.Topics[2].Bytes()).Hex(), new(big.Int).SetBytes(log.Data))
								index, err := env.txEventTbl.InsertEvent(txEvent, true)
								if err != nil {
									panic(err)
								}
								txEvent.Index = index
								//labs.Logger.WithFields(txEvent.LogFields()).Info("inserted transfer event")

								_, err = env.accRegistry.ProcessTransferEvent(txEvent)
								if err != nil {
									panic(err)
								}

								//for _, acc := range accs {
								//	labs.Logger.WithFields(acc.LogFields()).Info("update acc")
								//}

							}
						}
					}
				}
				block++
				/*
					prog := int(float64(block)/float64(latestBlock)*1000000) - 1
					pb1.Current = prog
					pb1.Increment()
				*/
			}
		}
	}
	return nil
}
