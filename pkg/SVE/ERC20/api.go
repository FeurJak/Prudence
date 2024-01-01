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
	FetchERC20Meta(tokenAddr common.Address, baseTokenAddr common.Address, factoryAddr common.Address) (meta *ERC20Meta, err error)
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

type erc20API struct {
	evm    evm
	logger logrus.Ext1FieldLogger
}

func NewAPI(evm evm, logger logrus.Ext1FieldLogger) ERC20API {
	return &erc20API{
		evm:    evm,
		logger: logger,
	}
}
