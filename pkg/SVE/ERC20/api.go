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

package ERC20

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"
)

type ERC20API interface {
	FetchERC20Meta(tokenAddr common.Address, baseTokenAddr common.Address, factoryAddr common.Address, routerAddr common.Address) (meta *ERC20Meta, err error)
}

type uniV2API interface {
	NewUniV2Factory(common.Address, common.Address) (interface{}, error)
	NewUniV2Pair(pairAddr common.Address, callerAddr common.Address) (interface{}, error)
	NewUniv2Router(routerAddr common.Address, callerAddr common.Address) (interface{}, error)
}

type uniV2Factory interface {
	GetPair(common.Address, common.Address) common.Address
}

type uniV2Router interface {
	RouterGetAmountOut(amountIn *big.Int, reserveIn *big.Int, reserveOut *big.Int) (amountOut *big.Int, err error)
	RouterSwapExactTokensForTokensSupportingFeeOnTransferTokens(amountIn *big.Int, amountOutMin *big.Int, path []common.Address, to common.Address, deadline *big.Int) error
}

type univ2Pair interface {
	Token0() common.Address
	GetReserves() (*big.Int, *big.Int)
	Sync() (err error)
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
	Time() uint64
}

type erc20API struct {
	evm      evm
	logger   logrus.Ext1FieldLogger
	univ2API uniV2API
}

func NewAPI(evm evm, logger logrus.Ext1FieldLogger, univ2API interface{}) ERC20API {
	return &erc20API{
		evm:      evm,
		logger:   logger,
		univ2API: univ2API.(uniV2API),
	}
}
