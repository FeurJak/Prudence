package main

import (
	"math/big"
	"time"

	labs "github.com/Prudence/labs"
	"github.com/sirupsen/logrus"

	erc20 "github.com/Prudence/pkg/SVE/ERC20"
	"github.com/Prudence/pkg/SVE/UNIV2"
	precog "github.com/Prudence/pkg/precog"
	sveVM "github.com/Prudence/pkg/vms/sve_vm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
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

	tokenAddr := common.HexToAddress("0x1673AB963C825402596F63e3d1Ef2c2966aa5340")
	baseTokenAddr := common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")
	factoryAddr := common.HexToAddress("0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f")
	routerAddr := common.HexToAddress("0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D")

	// Get Latest State
	state, header, err := psi.CurrentStateAndHeader()
	if state == nil || header == nil {
		panic("failed to get latest state")
	}

	// prep the EVM
	evm := sveVM.NewEVM(
		sveVM.NewEVMBlockContext(header, psi),
		sveVM.NewEVMTxContext(&core.Message{
			From:     tokenAddr,
			To:       &tokenAddr,
			GasPrice: big.NewInt(1),
		}), state, params.MainnetChainConfig, sveVM.Config{})

	// prepare univ2 API
	univ2API := UNIV2.NewAPI(evm, labs.Logger)

	// prepare ERC20 API
	erc20API := erc20.NewAPI(evm, labs.Logger, univ2API)

	// Fetch ERC20 Meta
	now := time.Now()
	meta, err := erc20API.FetchERC20Meta(tokenAddr, baseTokenAddr, factoryAddr, routerAddr)
	if err != nil {
		panic("failed to fetch ERC20 Meta")
	}

	labs.Logger.WithFields(logrus.Fields{
		"time":          time.Since(now),
		"Add":           meta.Addr,
		"Name":          meta.Name,
		"Owner":         meta.Owner,
		"BlockDelay":    meta.BlockDelay,
		"BuyTax":        meta.BuyTax,
		"SellTax":       meta.SellTax,
		"MaxWallet":     meta.MaxWalletSize,
		"MaxTx":         meta.MaxTxAmount,
		"Symbol":        meta.Symbol,
		"Decimals":      meta.Decimals,
		"TotalSupply":   meta.TotalSupply,
		"UniV2Pair":     meta.PairAddress,
		"TokenIsTokenA": meta.TokenIsTokenA,
		"Ra":            meta.ReserveA,
		"Rb":            meta.ReserveB,
	}).Info("got ERC20 Meta")

	//labs.Logger.Infof("ERC20 Meta: %+v", meta)
	return nil
}
