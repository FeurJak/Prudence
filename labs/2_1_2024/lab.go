package main

import (
	"math/big"
	"time"

	labs "github.com/Prudence/labs"

	erc20 "github.com/Prudence/pkg/SVE/ERC20"
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

	// prepare ERC20 API
	erc20API := erc20.NewAPI(evm)
	meta, err := erc20API.FetchERC20Meta(tokenAddr)
	if err != nil {
		panic("failed to fetch ERC20 Meta")
	}

	labs.Logger.Info("ERC20 Meta: ", meta)
	return nil
}
