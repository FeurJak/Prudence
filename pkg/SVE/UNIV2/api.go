/*

          _____                    _____                    _____
         /\    \                  /\    \                  /\    \
        /::\    \                /::\____\                /::\    \
       /::::\    \              /:::/    /               /::::\    \
      /::::::\    \            /:::/    /               /::::::\    \
     /:::/\:::\    \          /:::/    /               /:::/\:::\    \
    /:::/__\:::\    \        /:::/____/               /:::/__\:::\    \
    \:::\   \:::\    \       |::|    |               /::::\   \:::\    \
  ___\:::\   \:::\    \      |::|    |     _____    /::::::\   \:::\    \
 /\   \:::\   \:::\    \     |::|    |    /\    \  /:::/\:::\   \:::\    \
/::\   \:::\   \:::\____\    |::|    |   /::\____\/:::/__\:::\   \:::\____\
\:::\   \:::\   \::/    /    |::|    |  /:::/    /\:::\   \:::\   \::/    /
 \:::\   \:::\   \/____/     |::|    | /:::/    /  \:::\   \:::\   \/____/
  \:::\   \:::\    \         |::|____|/:::/    /    \:::\   \:::\    \
   \:::\   \:::\____\        |:::::::::::/    /      \:::\   \:::\____\
    \:::\  /:::/    /        \::::::::::/____/        \:::\   \::/    /
     \:::\/:::/    /          ~~~~~~~~~~               \:::\   \/____/
      \::::::/    /                                     \:::\    \
       \::::/    /                                       \:::\____\
        \::/    /                                         \::/    /
         \/____/                                           \/____/

	- Sustainable Virtual Economies (SVE).
*/

package UNIV2

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/sirupsen/logrus"
)

type UNIV2API interface {
	NewUniV2Factory(factoryAddr common.Address, callerAddr common.Address) (interface{}, error)
}

type contractCode interface {
	SetCallerAddress(callerAddr common.Address)
}

type evmInterpreter interface {
	LiteRun(contract interface{}, input []byte, readOnly bool) (ret []byte, d interface{}, err error)
	ScanFor(contract interface{}, input []byte, readOnly bool, value *uint256.Int) (loc uint64, res []byte, err error)
	ScanForMultiple(contract interface{}, input []byte, readOnly bool, values map[string]bool) (loc map[string][]uint64, res []byte, err error)
	RunWithOverride(contract interface{}, input []byte, readOnly bool, overrides interface{}, dCfg interface{}) (ret []byte, dRes interface{}, err error)
}

type evm interface {
	SnapshotDB() int
	InfGas(bool) // disable or enable infinite gas limit
	Interpreter() interface{}
	GetCode(common.Address) []byte
	GetCodeHash(common.Address) common.Hash
	NewContract(common.Address, common.Address, *big.Int, uint64, []byte) interface{}
	IncrementBlockNoBy(n *big.Int)
	BlockNumber() *big.Int
}

type univ2API struct {
	evm    evm
	logger logrus.Ext1FieldLogger
}

func NewAPI(evm evm, logger logrus.Ext1FieldLogger) UNIV2API {
	return &univ2API{
		evm:    evm,
		logger: logger,
	}
}
