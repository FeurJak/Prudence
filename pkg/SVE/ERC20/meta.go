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
	"fmt"
	"math/big"

	"github.com/Prudence/pkg/SVE/UNIV2"
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

type uniV2API interface {
	NewUniV2Factory(common.Address, common.Address) (interface{}, error)
}

type uniV2Factory interface {
	GetPair(common.Address, common.Address) common.Address
}

func (a *erc20API) FetchERC20Meta(tokenAddr common.Address, baseTokenAddr common.Address, factoryAddr common.Address) (meta *ERC20Meta, err error) {
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

	// Get Factory
	univ2API := UNIV2.NewAPI(a.evm, a.logger)

	v2Factory, err := univ2API.NewUniV2Factory(factoryAddr, rand_wallet)
	if err != nil {
		return nil, err
	}

	// Query the Factory for the Pair (Token <-> WETH/USDC/USDT/DAI)
	token.meta.PairAddress = v2Factory.(uniV2Factory).GetPair(tokenAddr, baseTokenAddr)
	if token.meta.PairAddress == (common.Address{}) {
		token.meta.Err = fmt.Sprintf("no uniswap pair found for token %s", tokenAddr.Hex())
		return token.meta, nil
	}

	/*
		// Get Pair
		pair, err := NewUniV2Pair(token.meta.PairAddress, rand_wallet, d.evm)
		if err != nil {
			return nil, err
		}

		// Get Token 0
		if pair.token0() == tokenAddr {
			token.meta.TokenIsTokenA = true
		}

		// Get Reserves
		token.meta.ReserveA, token.meta.ReserveB = pair.getReserves()
		if token.meta.ReserveA == nil || token.meta.ReserveB == nil {
			return token.meta, fmt.Errorf("failed to get reserves for pair %s", token.meta.PairAddress.Hex())
		}
	*/

	return token.meta, nil
}
