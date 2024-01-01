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

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

	BuyTax  *big.Int
	SellTax *big.Int

	HasMaxTx      bool
	HasMaxWallet  bool
	MaxWalletSize *big.Int
	MaxTxAmount   *big.Int

	PairAddress   common.Address
	TokenIsTokenA bool // direction of the pair
	ReserveA      *big.Int
	ReserveB      *big.Int

	Err string
}

func (a *erc20API) FetchERC20Meta(tokenAddr common.Address, baseTokenAddr common.Address, factoryAddr common.Address, routerAddr common.Address) (meta *ERC20Meta, err error) {
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

	v2Factory, err := a.univ2API.NewUniV2Factory(factoryAddr, rand_wallet)
	if err != nil {
		return nil, err
	}

	// Query the Factory for the Pair (Token <-> WETH/USDC/USDT/DAI)
	token.meta.PairAddress = v2Factory.(uniV2Factory).GetPair(tokenAddr, baseTokenAddr)
	if token.meta.PairAddress == (common.Address{}) {
		token.meta.Err = fmt.Sprintf("no uniswap pair found for token %s", tokenAddr.Hex())
		return token.meta, nil
	}

	// Get Pair
	pair, err := a.univ2API.NewUniV2Pair(token.meta.PairAddress, rand_wallet)
	if err != nil {
		return nil, err
	}

	// Get Token 0
	if pair.(univ2Pair).Token0() == tokenAddr {
		token.meta.TokenIsTokenA = true
	}

	// Get Reserves
	token.meta.ReserveA, token.meta.ReserveB = pair.(univ2Pair).GetReserves()
	if token.meta.ReserveA == nil || token.meta.ReserveB == nil {
		return token.meta, fmt.Errorf("failed to get reserves for pair %s", token.meta.PairAddress.Hex())
	}

	token.reference = new(execRef)

	// Get Router
	v2Router, err := a.univ2API.NewUniv2Router(routerAddr, rand_wallet)
	if err != nil {
		return nil, err
	}

	token.meta.MaxWalletSize = big.NewInt(0).Set(token.meta.TotalSupply)
	token.meta.MaxTxAmount = big.NewInt(0).Set(token.meta.TotalSupply)

	largeAmount, _ := big.NewInt(0).SetString("1000000000000000000000000000000", 10)
	largeAmount_uint256 := uint256.NewInt(0)
	largeAmount_uint256.SetFromBig(largeAmount)

	token.SetCallerAddress(token.meta.PairAddress)

	poolBalance := token.balanceOfUint256(token.meta.PairAddress)
	testAmount := uint256.NewInt(0).Div(poolBalance, uint256.NewInt(1000))

	/*
		Detect Block Delay
	*/
	token.SetCallerAddress(token.meta.PairAddress)
	if success, err := token.approve(token.meta.PairAddress, poolBalance.ToBig()); err == nil && success {
		token.detectBlockDelay(token.meta.PairAddress, rand_wallet)
	}

	/*
		Buy Tax.
		1. Transfer some base Token to the RANDOM_WALLET via the baseToken Pair.
		2. The amount of base Token to buy is 0.05% of the Base Token Reserve in the Target Token Pair.
		3. Calculate expected amount for buying Target Token.
		4. Buy Target Token with base Token & get actual amounts out.
		5. Calculate Buy tax.
	*/
	baseToken, err := newErc20Token(baseTokenAddr, rand_wallet, a.evm)
	if err != nil {
		return nil, err
	}

	baseToken.SetCallerAddress(token.meta.PairAddress)

	var baseDivider int64 = 200000
	baseTokenAmount := big.NewInt(0)
	if token.meta.TokenIsTokenA {
		baseTokenAmount = big.NewInt(0).Quo(token.meta.ReserveB, big.NewInt(baseDivider))
	} else {
		baseTokenAmount = big.NewInt(0).Quo(token.meta.ReserveA, big.NewInt(baseDivider))
	}

	// Load up Base Token.
	if success, err := baseToken.approve(token.meta.PairAddress, baseTokenAmount); err == nil && success {
		if success, err := baseToken.TransferFrom(token.meta.PairAddress, rand_wallet, baseTokenAmount); err == nil && success {
			baseTokenBalance := baseToken.balanceOfUint256(rand_wallet)

			a.logger.WithFields(logrus.Fields{
				"balance": baseTokenBalance,
			}).Info("base token balance")

			// Purchase Target Token with base Token.

			// Calculate Expected Amount
			pair.(univ2Pair).Sync()
			new_Ra, new_Rb := pair.(univ2Pair).GetReserves()

			_expected_amounts_out := big.NewInt(0)
			_actual_amounts_out := big.NewInt(0)
			if !token.meta.TokenIsTokenA {
				_expected_amounts_out, err = v2Router.(uniV2Router).RouterGetAmountOut(baseTokenBalance.ToBig(), new_Ra, new_Rb)
			} else {
				_expected_amounts_out, err = v2Router.(uniV2Router).RouterGetAmountOut(baseTokenBalance.ToBig(), new_Rb, new_Ra)
			}
			if err != nil || _expected_amounts_out == nil || _expected_amounts_out.Cmp(big.NewInt(0)) == 0 {
				return token.meta, errors.Wrap(err, "failed to get expected amounts out")
			}

			// Call Router to perform swap.
			baseToken.SetCallerAddress(rand_wallet)
			if success, err := baseToken.approve(routerAddr, baseTokenBalance.ToBig()); err == nil && success {
				baseToken.evm.IncrementBlockNoBy(token.meta.BlockDelay)
				if err = v2Router.(uniV2Router).RouterSwapExactTokensForTokensSupportingFeeOnTransferTokens(baseTokenBalance.ToBig(), big.NewInt(1), []common.Address{baseTokenAddr, tokenAddr}, rand_wallet, big.NewInt(int64(a.evm.Time())+1)); err != nil {
					return token.meta, fmt.Errorf("failed to swap tokens: %+v", err)
				}
				token.meta.CanBuy = true
				_actual_amounts_out = token.balanceOfUint256(rand_wallet).ToBig()

				// Calculate Buy Tax
				// 100 - (actual * 100 / expected)
				token.meta.BuyTax = calcTax(_actual_amounts_out, _expected_amounts_out)

				a.logger.WithFields(logrus.Fields{
					"In":       baseTokenBalance.ToBig(),
					"Expected": _expected_amounts_out,
					"Actual":   _actual_amounts_out,
					"CanBuy":   token.meta.CanBuy,
					"BuyTax":   token.meta.BuyTax,
				}).Info("bought")

			} else {
				return token.meta, errors.Wrap(err, "failed to approve base token")
			}
		}
	}

	/*
		Sell Tax. We only ever want to calculate the tax if we can work out the buyTax
	*/
	if token.meta.CanBuy {
		targetTokenBalance := token.balanceOfUint256(rand_wallet)
		// Sync up the Pool reserves with the latest balances
		pair.(univ2Pair).Sync()
		new_Ra, new_Rb := pair.(univ2Pair).GetReserves()

		// Calculate Expected Amount
		_expected_amounts_out := big.NewInt(0)
		_actual_amounts_out := big.NewInt(0)
		if token.meta.TokenIsTokenA {
			_expected_amounts_out, err = v2Router.(uniV2Router).RouterGetAmountOut(targetTokenBalance.ToBig(), new_Ra, new_Rb)
		} else {
			_expected_amounts_out, err = v2Router.(uniV2Router).RouterGetAmountOut(targetTokenBalance.ToBig(), new_Rb, new_Ra)
		}
		if err != nil || _expected_amounts_out == nil || _expected_amounts_out.Cmp(big.NewInt(0)) == 0 {
			return token.meta, errors.Wrap(err, "failed to get expected amounts out")
		}

		// Call Router to perform swap.
		token.SetCallerAddress(rand_wallet)
		token.evm.IncrementBlockNoBy(token.meta.BlockDelay)
		if success, err := token.approve(routerAddr, poolBalance.ToBig()); err == nil && success {
			if err = v2Router.(uniV2Router).RouterSwapExactTokensForTokensSupportingFeeOnTransferTokens(
				targetTokenBalance.ToBig(), big.NewInt(1),
				[]common.Address{tokenAddr, baseTokenAddr}, rand_wallet,
				big.NewInt(int64(a.evm.Time())+1)); err != nil {
				return token.meta, errors.Wrap(err, "failed to swap tokens")
			}
			token.meta.CanSell = true
			_actual_amounts_out = baseToken.balanceOfUint256(rand_wallet).ToBig()

			// Calculate Sell Tax
			token.meta.SellTax = calcTax(_actual_amounts_out, _expected_amounts_out)

			a.logger.WithFields(logrus.Fields{
				"In":       targetTokenBalance.ToBig(),
				"Expected": _expected_amounts_out,
				"Actual":   _actual_amounts_out,
				"CanSell":  token.meta.CanSell,
				"SellTax":  token.meta.SellTax,
			}).Info("sold")
		}
	}

	token.SetCallerAddress(token.meta.PairAddress)
	if success, err := token.approve(token.meta.PairAddress, largeAmount); err == nil && success {
		token.detectBlockDelay(token.meta.PairAddress, rand_wallet)
		token.evm.IncrementBlockNoBy(token.meta.BlockDelay)
		if success, err := token.TransferFrom(token.meta.PairAddress, rand_wallet, testAmount.ToBig()); err == nil && success {
			//token.meta.CanBuy = true
			allowance, err := token.allowanceUint256(token.meta.PairAddress, token.meta.PairAddress)
			if err != nil {
				return token.meta, errors.Wrap(err, "failed to get allowance")
			}
			poolBalance := token.balanceOfUint256(token.meta.PairAddress)
			walletBalance := token.balanceOfUint256(rand_wallet)
			tokenBalance := token.balanceOfUint256(token.meta.Addr)

			// Pop some tokens into Wallet to trace the necessary pCs.
			token.evm.IncrementBlockNoBy(token.meta.BlockDelay)
			if locations, _, err := token.TransferFromAndScan(token.meta.PairAddress, rand_wallet, big.NewInt(1), map[string]bool{
				allowance.String():     true,
				poolBalance.String():   true,
				walletBalance.String(): true,
				tokenBalance.String():  true,
			}); err == nil {

				// Do another Transfer but dump stack values
				// TODO: This is a hacky way to get the stack values, but it works.
				// Get the stack values from another transferFrom call, because now RANDOM_WALLET is populated with unique balance.

				if pc, ok := locations[allowance.String()]; ok {
					token.reference.allowancePC = pc[0]

					a.logger.WithFields(logrus.Fields{
						"pc":        token.reference.allowancePC,
						"allowance": allowance.ToBig(),
					}).Info("allowance PC")

				} else {
					return token.meta, fmt.Errorf("failed to get allowance pc, allowance: %+v", allowance)
				}

				if pc, ok := locations[poolBalance.String()]; ok {
					token.reference.poolBalancePc = pc[0]

					a.logger.WithFields(logrus.Fields{
						"pc":      token.reference.poolBalancePc,
						"balance": poolBalance.ToBig(),
					}).Info("poolBalance")

				} else {
					a.logger.WithFields(logrus.Fields{
						"pc":      token.reference.poolBalancePc,
						"balance": poolBalance.ToBig(),
						"err":     "failed to get pool balance",
					}).Error("poolBalance")
				}

				if pc, ok := locations[walletBalance.String()]; ok {
					token.reference.walletBalancePC = pc[0]

					a.logger.WithFields(logrus.Fields{
						"pc":      token.reference.walletBalancePC,
						"balance": walletBalance.ToBig(),
					}).Info("walletBalance")

				} else {
					return token.meta, fmt.Errorf("failed to get walletBalance pc, walletBalance: %+v", walletBalance)
				}

				if pc, ok := locations[tokenBalance.String()]; ok {
					token.reference.tokenBalancePC = pc[0]

					a.logger.WithFields(logrus.Fields{
						"pc":      token.reference.tokenBalancePC,
						"balance": tokenBalance.ToBig(),
					}).Info("tokenBalance")

				} else {
					return token.meta, fmt.Errorf("failed to get tokenBalance pc, tokenBalance: %+v", tokenBalance)
				}

				// Now we get the MaxWalletSize
				token.getMaxWallet(token.meta.PairAddress, rand_wallet)
				token.meta.MaxTxAmount = big.NewInt(0).Set(token.meta.MaxWalletSize)

				// Then we get maxTX Amount.
				if success, err := token.approve(token.meta.PairAddress, largeAmount); err == nil && success {
					token.getMaxTxAmount(token.meta.PairAddress, rand_wallet)
				}
			} else {
				return token.meta, errors.Wrap(err, "failed to transfer from")
			}
		} else {
			return token.meta, errors.Wrap(err, "failed to transfer from")
		}
	} else {
		return token.meta, errors.Wrap(err, "failed to approve token")
	}

	return token.meta, nil
}

// Simple func containing tax calculation formula
func calcTax(actualAmountsOut, expectedAmountsOut *big.Int) *big.Int {
	hundred := big.NewInt(100)

	multipliedAmounts := big.NewInt(0).Mul(actualAmountsOut, hundred)
	dividedAmounts := big.NewInt(0).Div(multipliedAmounts, expectedAmountsOut)
	subtractedAmounts := big.NewInt(0).Sub(hundred, dividedAmounts)

	return subtractedAmounts
}
