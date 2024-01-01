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
)

var (
	rand_wallet   = common.HexToAddress("0xD5F4A3421B1A3188154fbC2B8a3378143cB4C053")
	rand_wallet_b = common.HexToAddress("0x0131B23d141fDCb29dE74D7dc3028676A17d7854")
)

type ERC20Meta struct {
	Addr        common.Address
	Name        string
	Symbol      string
	Decimals    uint8
	TotalSupply *big.Int

	Owner common.Address

	CanBuy  bool
	CanSell bool

	BlockDelayEnabled bool
	BlockDelay        *big.Int

	BuyTax      *big.Int
	SellTax     *big.Int
	TransferFee *big.Int

	HasMaxTx      bool
	HasMaxWallet  bool
	MaxWalletSize *big.Int
	MaxTxAmount   *big.Int

	PairAddress   common.Address
	TokenIsTokenA bool // direction of the pair
	ReserveA      *big.Int
	ReserveB      *big.Int
	PriceToWeth   *big.Float
	PriceInWei    *big.Float

	Err string
}

func (a *erc20API) FetchERC20Meta(tokenAddr common.Address) (meta *ERC20Meta, err error) {
	//snapshot := d.evm.SnapshotDB()
	a.evm.InfGas(true)

	token, err := newErc20Token(tokenAddr, rand_wallet, a.evm)
	if err != nil {
		return nil, err
	}

	// Get Basic ERC20 Meta
	token.meta.Name = token.name()
	token.meta.Symbol = token.symbol()
	token.meta.Decimals = token.decimals()
	token.meta.TotalSupply = token.totalSupply()
	token.meta.Owner = token.owner()

	return token.meta, nil
}
